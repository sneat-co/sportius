package facade4sportius

import (
	sportius "github.com/sneat-co/ext-sportius/backend"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

func profileRecord(userID string, sportID sportius.SportID) models4sportius.PersonalProfileRecord {
	return models4sportius.PersonalProfileRecord{
		SpaceID: "personal-" + userID,
		UserID:  userID,
		Sports: map[sportius.SportID]sportius.PersonalSport{
			sportID: {
				SportID:    sportID,
				Visibility: sportius.VisibilityPrivate,
			},
		},
	}
}

func personalProfileRef(userID string) PersonalProfileRef {
	return PersonalProfileRef{SpaceID: "personal-" + userID, UserID: userID}
}
