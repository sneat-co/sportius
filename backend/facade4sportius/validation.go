package facade4sportius

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	sportius "github.com/sneat-co/ext-sportius/backend"
)

func validateActor(actorUserID string) error {
	if strings.TrimSpace(actorUserID) == "" {
		return invalidf("authenticated actor user ID is required")
	}
	return nil
}

func validateRequestID(requestID string) error {
	if strings.TrimSpace(requestID) == "" {
		return invalidf("request ID is required")
	}
	if len(requestID) > 200 {
		return invalidf("request ID is too long")
	}
	return nil
}

func validateName(kind, value string) (string, error) {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "", invalidf("%s name is required", kind)
	}
	if len([]rune(value)) > 160 {
		return "", invalidf("%s name is too long", kind)
	}
	return value, nil
}

func validateSport(id sportius.SportID) error {
	switch id {
	case sportius.SportBasketball,
		sportius.SportFootball,
		sportius.SportGaelic,
		sportius.SportHockey,
		sportius.SportRugby,
		sportius.SportTennis,
		sportius.SportSwimming,
		sportius.SportVolleyball,
		sportius.SportOther:
		return nil
	default:
		return invalidf("unsupported sport %q", id)
	}
}

func validateSports(ids []sportius.SportID) ([]sportius.SportID, error) {
	result := make([]sportius.SportID, 0, len(ids))
	seen := make(map[sportius.SportID]bool, len(ids))
	for _, id := range ids {
		if err := validateSport(id); err != nil {
			return nil, err
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result, nil
}

func validateRoles(ids []sportius.RoleID, scope sportius.RoleScope) ([]sportius.RoleID, error) {
	result := make([]sportius.RoleID, 0, len(ids))
	seen := make(map[sportius.RoleID]bool, len(ids))
	for _, id := range ids {
		definition, ok := roleDefinition(id)
		if !ok {
			return nil, invalidf("unsupported role %q", id)
		}
		if !hasScope(definition.Scopes, scope) {
			return nil, invalidf("role %q is not valid for %s scope", id, scope)
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result, nil
}

func roleDefinition(id sportius.RoleID) (sportius.RoleDefinition, bool) {
	for _, definition := range sportius.RoleCatalog {
		if definition.ID == id {
			return definition, true
		}
	}
	return sportius.RoleDefinition{}, false
}

func hasScope(scopes []sportius.RoleScope, expected sportius.RoleScope) bool {
	for _, scope := range scopes {
		if scope == expected {
			return true
		}
	}
	return false
}

func validVisibility(value sportius.ProfileVisibility) bool {
	return value == sportius.VisibilityPrivate || value == sportius.VisibilityPublic || value == sportius.VisibilityHidden
}

func validateGender(value sportius.GenderCategory) (sportius.GenderCategory, error) {
	if value == "" {
		return sportius.GenderUnspecified, nil
	}
	switch value {
	case sportius.GenderUnspecified, sportius.GenderMale, sportius.GenderFemale, sportius.GenderMixed, sportius.GenderOther:
		return value, nil
	default:
		return "", invalidf("unsupported gender category %q", value)
	}
}

func validateJoinPolicy(value sportius.JoinPolicy) (sportius.JoinPolicy, error) {
	if value == "" {
		return sportius.JoinPolicyOpen, nil
	}
	switch value {
	case sportius.JoinPolicyOpen, sportius.JoinPolicyApprovalRequired, sportius.JoinPolicyInviteOnly:
		return value, nil
	default:
		return "", invalidf("unsupported join policy %q", value)
	}
}

func validateAge(value *sportius.AgeRange) (*sportius.AgeRange, error) {
	if value == nil {
		return nil, nil
	}
	result := *value
	result.Label = strings.TrimSpace(result.Label)
	if result.MinAge != nil {
		minAge := *result.MinAge
		if minAge < 0 || minAge > 120 {
			return nil, invalidf("minimum age must be between 0 and 120")
		}
		result.MinAge = &minAge
	}
	if result.MaxAge != nil {
		maxAge := *result.MaxAge
		if maxAge < 0 || maxAge > 120 {
			return nil, invalidf("maximum age must be between 0 and 120")
		}
		result.MaxAge = &maxAge
	}
	if result.MinAge != nil && result.MaxAge != nil && *result.MinAge > *result.MaxAge {
		return nil, invalidf("minimum age must not exceed maximum age")
	}
	if result.MinAge == nil && result.MaxAge == nil && result.Label == "" {
		return nil, nil
	}
	return &result, nil
}

func validateLocation(value *sportius.LocationHint) (*sportius.LocationHint, error) {
	if value == nil {
		return nil, nil
	}
	result := *value
	result.Locality = strings.Join(strings.Fields(result.Locality), " ")
	result.Region = strings.Join(strings.Fields(result.Region), " ")
	result.CountryID = strings.ToUpper(strings.TrimSpace(result.CountryID))
	result.TogetheredSpotID = strings.TrimSpace(result.TogetheredSpotID)
	if result.Point != nil {
		point := *result.Point
		if point.Latitude < -90 || point.Latitude > 90 {
			return nil, invalidf("latitude must be between -90 and 90")
		}
		if point.Longitude < -180 || point.Longitude > 180 {
			return nil, invalidf("longitude must be between -180 and 180")
		}
		result.Point = &point
	}
	if result.Locality == "" && result.Region == "" && result.CountryID == "" && result.Point == nil && result.TogetheredSpotID == "" {
		return nil, nil
	}
	return &result, nil
}

func validateMedia(value *sportius.MediaRef) (*sportius.MediaRef, error) {
	if value == nil {
		return nil, nil
	}
	result := *value
	result.FileID = strings.TrimSpace(result.FileID)
	result.Kind = strings.TrimSpace(result.Kind)
	if result.FileID == "" && result.Kind == "" {
		return nil, nil
	}
	if result.FileID == "" {
		return nil, invalidf("media file ID is required")
	}
	if result.Kind == "" {
		result.Kind = "photo"
	}
	return &result, nil
}

func normaliseName(value string) string {
	var builder strings.Builder
	previousSpace := true
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			previousSpace = false
			continue
		}
		if !previousSpace {
			builder.WriteByte(' ')
			previousSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

func hasRole(roles []sportius.RoleID, expected sportius.RoleID) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

func hasSport(sports []sportius.SportID, expected sportius.SportID) bool {
	for _, sport := range sports {
		if sport == expected {
			return true
		}
	}
	return false
}

func mergeRoles(first, second []sportius.RoleID) []sportius.RoleID {
	result := append([]sportius.RoleID(nil), first...)
	seen := make(map[sportius.RoleID]bool, len(first)+len(second))
	for _, role := range first {
		seen[role] = true
	}
	for _, role := range second {
		if !seen[role] {
			seen[role] = true
			result = append(result, role)
		}
	}
	return result
}

func mergeSports(first, second []sportius.SportID) []sportius.SportID {
	result := append([]sportius.SportID(nil), first...)
	seen := make(map[sportius.SportID]bool, len(first)+len(second))
	for _, sport := range first {
		seen[sport] = true
	}
	for _, sport := range second {
		if !seen[sport] {
			seen[sport] = true
			result = append(result, sport)
		}
	}
	return result
}

func sameRoles(first, second []sportius.RoleID) bool {
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}

func rolesIncludeStaff(roles []sportius.RoleID) bool {
	for _, role := range roles {
		definition, ok := roleDefinition(role)
		if ok && definition.ImpliesStaff {
			return true
		}
	}
	return false
}

func commandFingerprint(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("Sportius command fingerprint: %v", err))
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func commandRequestKey(operation, actorUserID, requestID string) string {
	return operation + ":" + commandFingerprint(struct {
		ActorUserID string
		RequestID   string
	}{ActorUserID: actorUserID, RequestID: requestID})[:32]
}

func fingerprintRoles(values []sportius.RoleID) []sportius.RoleID {
	result := append([]sportius.RoleID(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func fingerprintSports(values []sportius.SportID) []sportius.SportID {
	result := append([]sportius.SportID(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func sortTeamBriefs(values []sportius.TeamBrief) {
	sort.Slice(values, func(i, j int) bool {
		left, right := normaliseName(values[i].Name), normaliseName(values[j].Name)
		if left == right {
			return values[i].SpaceID < values[j].SpaceID
		}
		return left < right
	})
}

func sortClubBriefs(values []sportius.ClubBrief) {
	sort.Slice(values, func(i, j int) bool {
		left, right := normaliseName(values[i].Name), normaliseName(values[j].Name)
		if left == right {
			return values[i].SpaceID < values[j].SpaceID
		}
		return left < right
	})
}
