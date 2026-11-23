package adult

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// AgeAssurance is the adults-only eligibility check. The returned string is
// the assurance mode identifier reported by AdultStatus.ageAssurance; it is
// never an assessment of a person or a compatibility score.
//
// SECURITY / PRD §7.14: the default implementation denies everyone. A
// self-entered birthday is NOT sufficient for a high-risk dating feature and
// is deliberately not stored anywhere in this schema. A real, jurisdiction-
// appropriate age-assurance provider must be integrated and validated before
// this feature can be enabled for real users.
type AgeAssurance interface {
	Eligible(userID int64) (bool, string)
}

const (
	assuranceNone            = "none"
	assuranceDevelopmentStub = "development_stub"

	// envAgeAssurance selects the non-production stub. Any other value (including
	// an empty one) leaves the deny-all default in place.
	envAgeAssurance = "ADULT_AGE_ASSURANCE"

	assuranceQueryTimeout = 3 * time.Second
)

// denyAll is the safe default: nobody is eligible and the reported mode is
// "none". No provider has been configured, so no access can be granted.
type denyAll struct{}

func (denyAll) Eligible(int64) (bool, string) { return false, assuranceNone }

// developmentStub is a LOCAL-ONLY stand-in for a real provider. It reads the
// verification row written exclusively by POST /adult/dev/verify. It is never
// constructed outside development, and it is not an age check: it only exists
// so the disabled-by-default scaffold can be exercised in tests.
type developmentStub struct {
	db *sql.DB
}

func (s developmentStub) Eligible(userID int64) (bool, string) {
	if s.db == nil || userID <= 0 {
		return false, assuranceDevelopmentStub
	}
	ctx, cancel := context.WithTimeout(context.Background(), assuranceQueryTimeout)
	defer cancel()
	var one int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM adult_verifications
		WHERE user_id = ? AND status = 'verified'
		  AND (expires_at IS NULL OR expires_at > datetime('now'))
		LIMIT 1`, userID).Scan(&one)
	return err == nil, assuranceDevelopmentStub
}

// newAgeAssurance selects the stub only when ALL three conditions hold:
// FEATURE_ADULT_DATING is enabled, APP_ENV is development, and
// ADULT_AGE_ASSURANCE=development_stub. Every other combination is deny-all.
func newAgeAssurance(d *shared.Deps) AgeAssurance {
	if d == nil || !d.Config.FeatureAdultDating || !isDevelopmentEnv(d.Config.AppEnv) {
		return denyAll{}
	}
	if strings.TrimSpace(os.Getenv(envAgeAssurance)) != assuranceDevelopmentStub {
		return denyAll{}
	}
	return developmentStub{db: d.DB}
}

func isDevelopmentEnv(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), "development")
}
