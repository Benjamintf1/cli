package v7action

import (
	"sort"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
)

// User represents a CLI user.
// This means that v7action.User has the same fields as a ccv3.User
type User ccv3.User

// CreateUser creates a new user in UAA and registers it with cloud controller.
func (actor Actor) CreateUser(username string, password string, origin string) (User, Warnings, error) {
	uaaUser, err := actor.UAAClient.CreateUser(username, password, origin)
	if err != nil {
		return User{}, nil, err
	}

	ccUser, ccWarnings, err := actor.CloudControllerClient.CreateUser(uaaUser.ID)

	return User(ccUser), Warnings(ccWarnings), err
}

// GetUser gets a user in UAA with the given username and (if provided) origin.
// It returns an error if no matching user is found.
// It returns an error if multiple matching users are found.
// NOTE: The UAA /Users endpoint used here requires admin scopes.
func (actor Actor) GetUser(username, origin string) (User, error) {
	uaaUsers, err := actor.UAAClient.ListUsers(username, origin)
	if err != nil {
		return User{}, err
	}

	if len(uaaUsers) == 0 {
		return User{}, actionerror.UserNotFoundError{Username: username, Origin: origin}
	}

	if len(uaaUsers) > 1 {
		var origins []string
		for _, user := range uaaUsers {
			origins = append(origins, user.Origin)
		}
		return User{}, actionerror.MultipleUAAUsersFoundError{Username: username, Origins: origins}
	}

	uaaUser := uaaUsers[0]

	v7actionUser := User{
		GUID:   uaaUser.ID,
		Origin: uaaUser.Origin,
	}
	return v7actionUser, nil
}

// DeleteUser
func (actor Actor) DeleteUser(userGuid string) (Warnings, error) {
	var allWarnings Warnings
	jobURL, ccWarningsDelete, err := actor.CloudControllerClient.DeleteUser(userGuid)
	allWarnings = Warnings(ccWarningsDelete)

	// If there is an error that is not a ResourceNotFoundError
	if _, ok := err.(ccerror.ResourceNotFoundError); !ok && err != nil {
		return allWarnings, err
	}

	ccWarningsPoll, err := actor.CloudControllerClient.PollJob(jobURL)
	allWarnings = append(allWarnings, Warnings(ccWarningsPoll)...)
	if err != nil {
		return allWarnings, err
	}

	_, err = actor.UAAClient.DeleteUser(userGuid)

	return allWarnings, err
}

func (actor Actor) UpdateUserPassword(userGUID string, oldPassword string, newPassword string) error {
	return actor.UAAClient.UpdatePassword(userGUID, oldPassword, newPassword)
}

func SortUsers(users []User) {
	sort.Slice(users, func(i, j int) bool {
		if users[i].PresentationName == users[j].PresentationName {

			if users[i].Origin == constant.DefaultOriginUaa || users[j].Origin == "" {
				return true
			}

			if users[j].Origin == constant.DefaultOriginUaa || users[i].Origin == "" {
				return false
			}

			return users[i].Origin < users[j].Origin
		}

		return users[i].PresentationName < users[j].PresentationName
	})
}

func GetHumanReadableOrigin(user User) string {
	if user.Origin == "" {
		return "client"
	}

	return user.Origin
}
