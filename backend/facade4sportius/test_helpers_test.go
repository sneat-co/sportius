package facade4sportius

import (
	sportius "github.com/sneat-co/ext-sportius/backend"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

func profileRecord(userID string, sportID sportius.SportID) models4sportius.PersonalProfileRecord {
	return models4sportius.PersonalProfileRecord{
		UserID: userID,
		Sports: map[sportius.SportID]sportius.PersonalSport{
			sportID: {
				SportID:    sportID,
				Visibility: sportius.VisibilityPrivate,
			},
		},
	}
}
