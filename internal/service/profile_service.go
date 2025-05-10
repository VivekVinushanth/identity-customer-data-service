package service

import (
	"context"
	"fmt"
	"github.com/wso2/identity-customer-data-service/internal/constants"
	"github.com/wso2/identity-customer-data-service/internal/database"
	errors "github.com/wso2/identity-customer-data-service/internal/errors"
	"github.com/wso2/identity-customer-data-service/internal/logger"
	"github.com/wso2/identity-customer-data-service/internal/models"
	"github.com/wso2/identity-customer-data-service/internal/repository"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func CreateOrUpdateProfile(event models.Event) (*models.Profile, error) {

	ctx := context.Background()
	postgresDB := database.GetPostgresInstance()
	conn, err := postgresDB.DB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DB connection: %w", err)
	}
	defer conn.Close() // ensures lock is released and session is closed

	// Create a lock tied to this connection
	lock := database.NewPostgresLock(conn)
	lockIdentifier := event.ProfileId

	//  Attempt to acquire the lock with retry
	var acquired bool
	for i := 0; i < constants.MaxRetryAttempts; i++ {
		acquired, err = lock.Acquire(lockIdentifier)
		if err != nil {
			return nil, fmt.Errorf("lock acquisition error for %s: %w", event.ProfileId, err)
		}
		if acquired {
			break
		}
		time.Sleep(constants.RetryDelay)
	}
	if !acquired {
		return nil, fmt.Errorf("could not acquire lock for profile %s after %d retries", event.ProfileId, constants.MaxRetryAttempts)
	}
	defer func() {
		_ = lock.Release(lockIdentifier) //  Always attempt to release
	}()

	//  Insert/update using standard DB (does not have to use same conn unless needed)
	profileRepo := repositories.NewProfileRepository(postgresDB.DB)
	profileToUpsert := models.Profile{
		ProfileId: event.ProfileId,
		ProfileHierarchy: &models.ProfileHierarchy{
			IsParent:    true,
			ListProfile: true,
		},
		IdentityAttributes: make(map[string]interface{}),
		Traits:             make(map[string]interface{}),
		ApplicationData:    []models.ApplicationData{},
	}

	if err := profileRepo.InsertProfile(profileToUpsert); err != nil {
		return nil, fmt.Errorf("failed to insert or update profile %s: %w", event.ProfileId, err)
	}

	profileFetched, errWait := waitForProfile(event.ProfileId, constants.MaxRetryAttempts, constants.RetryDelay)
	if errWait != nil || profileFetched == nil {
		return nil, fmt.Errorf("profile %s not visible after insert/update: %w", event.ProfileId, errWait)
	}

	return profileFetched, nil
}

// GetProfile retrieves a profile
func GetProfile(ProfileId string) (*models.Profile, error) {

	postgresDB := database.GetPostgresInstance()
	profileRepo := repositories.NewProfileRepository(postgresDB.DB)

	profile, _ := profileRepo.GetProfile(ProfileId)
	if profile == nil {
		clientError := errors.NewClientError(errors.ErrorMessage{
			Code:        errors.ErrProfileNotFound.Code,
			Message:     errors.ErrProfileNotFound.Message,
			Description: errors.ErrProfileNotFound.Description,
		}, http.StatusNotFound)
		return nil, clientError
	}

	if profile.ProfileHierarchy.IsParent {
		return profile, nil
	} else {
		// fetching merged master profile
		masterProfile, err := profileRepo.GetProfile(profile.ProfileHierarchy.ParentProfileID)
		// todo: app context should be restricted for apps that is requesting these

		masterProfile.ApplicationData, _ = profileRepo.FetchApplicationData(masterProfile.ProfileId)

		// building the hierarchy
		masterProfile.ProfileHierarchy.ChildProfiles, _ = profileRepo.FetchChildProfiles(masterProfile.ProfileId)
		masterProfile.ProfileHierarchy.ParentProfileID = masterProfile.ProfileId
		masterProfile.ProfileId = profile.ProfileId

		if err != nil {
			return nil, errors.NewServerError(errors.ErrWhileFetchingProfile, err)
		}
		if masterProfile == nil {
			logger.Debug("Master profile is unfortunately empty")
			return nil, nil
		}
		return masterProfile, nil
	}
}

// DeleteProfile removes a profile from MongoDB by `perma_id`
func DeleteProfile(ProfileId string) error {

	postgresDB := database.GetPostgresInstance()
	eventRepo := repositories.NewEventRepository(postgresDB.DB)
	profileRepo := repositories.NewProfileRepository(postgresDB.DB)

	// Fetch the existing profile before deletion
	profile, err := profileRepo.GetProfile(ProfileId)
	if profile == nil {
		logger.Info(fmt.Sprintf("Profile with profile_id: %s that is requested for deletion is not found",
			ProfileId))
		return nil
	}
	if err != nil {
		return errors.NewServerError(errors.ErrWhileFetchingProfile, err)
	}

	//  Delete related events
	if err := eventRepo.DeleteEventsByProfileId(ProfileId); err != nil {
		return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
	}

	if profile.ProfileHierarchy.IsParent {
		// fetching the child if its parent
		profile.ProfileHierarchy.ChildProfiles, _ = profileRepo.FetchChildProfiles(profile.ProfileId)
	}

	if profile.ProfileHierarchy.IsParent && len(profile.ProfileHierarchy.ChildProfiles) == 0 {
		// Delete the parent with no children
		err = profileRepo.DeleteProfile(ProfileId)
		if err != nil {
			return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
		}
		return nil
	}

	if profile.ProfileHierarchy.IsParent && len(profile.ProfileHierarchy.ChildProfiles) > 0 {
		//get all child profiles and delete
		for _, childProfile := range profile.ProfileHierarchy.ChildProfiles {
			profile, err := profileRepo.GetProfile(childProfile.ChildProfileId)
			if profile == nil {
				logger.Debug("Child profile with profile_id: %s that is being deleted is not found",
					childProfile.ChildProfileId)
				return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
			}
			if err != nil {
				return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
			}
			err = profileRepo.DeleteProfile(childProfile.ChildProfileId)
			if err != nil {
				return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
			}
		}
		// now delete master
		err = profileRepo.DeleteProfile(ProfileId)
		if err != nil {
			return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
		}
		return nil
	}

	// If it is a child profile, delete it
	if !(profile.ProfileHierarchy.IsParent) {
		parentProfile, err := profileRepo.GetProfile(profile.ProfileHierarchy.ParentProfileID)
		parentProfile.ProfileHierarchy.ChildProfiles, _ = profileRepo.FetchChildProfiles(parentProfile.ProfileId)

		if len(parentProfile.ProfileHierarchy.ChildProfiles) == 1 {
			// delete the parent as this is the only child
			err = profileRepo.DeleteProfile(profile.ProfileHierarchy.ParentProfileID)
			err = profileRepo.DeleteProfile(ProfileId)
			if err != nil {
				return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
			}
		} else {
			err = profileRepo.DetachChildProfileFromParent(profile.ProfileHierarchy.ParentProfileID, ProfileId)
			if err != nil {
				return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
			}
			err = profileRepo.DeleteProfile(ProfileId)
			if err != nil {
				return errors.NewServerError(errors.ErrWhileDeletingProfile, err)
			}
		}

	}

	return nil
}

func waitForProfile(profileID string, maxRetries int, retryDelay time.Duration) (*models.Profile, error) {

	var profile *models.Profile
	var lastErr error
	postgresDB := database.GetPostgresInstance()
	profileRepo := repositories.NewProfileRepository(postgresDB.DB)

	for i := 0; i < maxRetries; i++ {
		if i > 0 { // Only sleep on subsequent retries
			time.Sleep(retryDelay)
		}
		profile, lastErr = profileRepo.GetProfile(profileID) // Assuming GetProfile is a method on profileRepo
		if profile != nil {
			return profile, nil
		}
		if lastErr != nil {
			log.Print("waitForProfile: Error during fetch attempt", "profileId", profileID, "attempt", i+1, "error", lastErr)
			// Continue to retry, lastErr will be reported if all retries fail
		}
	}

	// logger.Error("waitForProfile: Profile not visible after all retries", "profileId", profileID, "attempts", maxRetries)
	if lastErr != nil {
		return nil, fmt.Errorf("profile %s not visible after %d retries, last error: %w", profileID, maxRetries, lastErr)
	}
	return nil, fmt.Errorf("profile %s not visible after %d retries", profileID, maxRetries)
}

// GetAllProfiles retrieves all profiles
func GetAllProfiles() ([]models.Profile, error) {

	postgresDB := database.GetPostgresInstance() // Your method to get *sql.DB wrapped or raw
	profileRepo := repositories.NewProfileRepository(postgresDB.DB)

	existingProfiles, err := profileRepo.GetAllProfiles()
	if err != nil {
		return nil, errors.NewServerError(errors.ErrWhileFetchingProfile, err)
	}
	if existingProfiles == nil {
		return []models.Profile{}, nil
	}

	// todo: app context should be restricted for apps that is requesting these

	var result []models.Profile
	for _, profile := range existingProfiles {
		if profile.ProfileHierarchy.IsParent {
			result = append(result, profile)
		} else {
			// Fetch master and assign current profile ID
			master, err := profileRepo.GetProfile(profile.ProfileHierarchy.ParentProfileID)
			if err != nil || master == nil {
				continue
			}

			master.ApplicationData, _ = profileRepo.FetchApplicationData(master.ProfileId)

			// building the hierarchy
			master.ProfileHierarchy.ChildProfiles, _ = profileRepo.FetchChildProfiles(master.ProfileId)
			master.ProfileId = profile.ProfileId
			master.ProfileHierarchy.ParentProfileID = master.ProfileId

			result = append(result, *master)
		}
	}
	return result, nil
}

// GetAllProfilesWithFilter handles fetching all profiles with filter
func GetAllProfilesWithFilter(filters []string) ([]models.Profile, error) {

	postgresDB := database.GetPostgresInstance()
	schemaRepo := repositories.NewProfileSchemaRepository(postgresDB.DB)
	profileRepo := repositories.NewProfileRepository(postgresDB.DB)

	rules, err := schemaRepo.GetProfileEnrichmentRules()
	if err != nil {
		return nil, errors.NewServerError(errors.ErrWhileFetchingProfileEnrichmentRules, err)
	}

	// Step 2: Build trait → valueType mapping
	propertyTypeMap := make(map[string]string)
	for _, rule := range rules {
		propertyTypeMap[rule.PropertyName] = rule.ValueType
	}

	// Step 3: Rewrite filters with correct parsed types
	var updatedFilters []string
	for _, f := range filters {
		parts := strings.SplitN(f, " ", 3)
		if len(parts) != 3 {
			continue
		}
		field, operator, rawValue := parts[0], parts[1], parts[2]
		valueType := propertyTypeMap[field]
		parsed := parseTypedValueForFilters(valueType, rawValue)

		// Prepare updated filter string
		var valueStr string
		switch v := parsed.(type) {
		case string:
			valueStr = v
		default:
			valueStr = fmt.Sprintf("%v", v)
		}
		updatedFilters = append(updatedFilters, fmt.Sprintf("%s %s %s", field, operator, valueStr))
	}

	// Step 4: Pass updated filters to repo
	existingProfiles, err := profileRepo.GetAllProfilesWithFilter(updatedFilters)
	if err != nil {
		return nil, errors.NewServerError(errors.ErrWhileFetchingProfile, err)
	}
	if existingProfiles == nil {
		existingProfiles = []models.Profile{}
	}

	// todo: app context should be restricted for apps that is requesting these

	var result []models.Profile
	for _, profile := range existingProfiles {
		if profile.ProfileHierarchy.IsParent {
			result = append(result, profile)
		} else {
			// Fetch master and assign current profile ID
			master, err := profileRepo.GetProfile(profile.ProfileHierarchy.ParentProfileID)
			if err != nil || master == nil {
				continue
			}

			master.ApplicationData, _ = profileRepo.FetchApplicationData(master.ProfileId)

			// building the hierarchy
			master.ProfileHierarchy.ChildProfiles, _ = profileRepo.FetchChildProfiles(master.ProfileId)
			master.ProfileId = profile.ProfileId
			master.ProfileHierarchy.ParentProfileID = master.ProfileId

			result = append(result, *master)
		}
	}
	return result, nil
}

func parseTypedValueForFilters(valueType string, raw string) interface{} {
	switch valueType {
	case "int":
		i, _ := strconv.Atoi(raw)
		return i
	case "float", "double":
		f, _ := strconv.ParseFloat(raw, 64)
		return f
	case "boolean":
		return raw == "true"
	case "string":
		return raw
	default:
		return raw
	}
}
