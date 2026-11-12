package store_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"bankpiggy/api/internal/testdb"
)

func invitationStore(t *testing.T, names ...string) *store.Store {
	t.Helper()
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", budget.Blank(names...)); err != nil {
		t.Fatal(err)
	}
	if err := database.BindIdentity(t.Context(), "household", "member-1", "one@example.com", "first-subject"); err != nil {
		t.Fatal(err)
	}
	return database
}

func setInvitation(t *testing.T, database *store.Store, hash string) {
	t.Helper()
	if err := database.SetInvitation(t.Context(), "household", hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func TestInvitationRotationExpirationAndRevocationInvalidateClaims(t *testing.T) {
	database := invitationStore(t, "First")
	setInvitation(t, database, "original-hash")
	preview, err := database.InvitationPreview(t.Context(), "household", "original-hash")
	if err != nil || preview.HouseholdName != "First" || preview.MemberName != "New person" {
		t.Fatalf("unexpected preview: %+v, %v", preview, err)
	}
	setInvitation(t, database, "rotated-hash")
	assertUnavailableInvitation(t, database, "original-hash")
	if err := database.SetInvitation(t.Context(), "household", "expired-hash", time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	assertUnavailableInvitation(t, database, "expired-hash")
	status, err := database.InvitationStatus(t.Context(), "household")
	if err != nil || status.Linked || status.ExpiresAt != nil {
		t.Fatalf("expired link is pending: %+v, %v", status, err)
	}
	setInvitation(t, database, "revoked-hash")
	if err := database.RevokeInvitation(t.Context(), "household"); err != nil {
		t.Fatal(err)
	}
	assertUnavailableInvitation(t, database, "revoked-hash")
}

func assertUnavailableInvitation(t *testing.T, database *store.Store, hash string) {
	t.Helper()
	if _, err := database.InvitationPreview(t.Context(), "household", hash); !errors.Is(err, store.ErrInvitationUnavailable) {
		t.Fatalf("unavailable invitation preview returned %v", err)
	}
	err := database.ClaimInvitation(t.Context(), "household", hash, "two@example.com", "second-subject", "Second")
	if !errors.Is(err, store.ErrInvitationUnavailable) {
		t.Fatalf("unavailable invitation claim returned %v", err)
	}
}

func TestInvitationClaimAddsOneBlankProfileAndPersistsIdentity(t *testing.T) {
	database := invitationStore(t, "First")
	setInvitation(t, database, "hash")
	if err := database.ClaimInvitation(t.Context(), "household", "hash", "two@example.com", "second-subject", "Second"); err != nil {
		t.Fatal(err)
	}
	saved, err := database.Read(t.Context(), "household")
	if err != nil {
		t.Fatal(err)
	}
	expected := budget.Blank("First", "Second")
	expected.Revision++
	if !reflect.DeepEqual(saved, expected) {
		t.Fatal("claim did not add exactly one blank profile")
	}
	identity, err := database.Identity(t.Context(), "household", "two@example.com", "second-subject")
	if err != nil || identity.MemberID != "member-2" || !identity.InvitationAccepted {
		t.Fatalf("accepted identity was not persisted: %+v %v", identity, err)
	}
	if err := database.ClaimInvitation(t.Context(), "household", "hash", "third@example.com", "third-subject", "Third"); !errors.Is(err, store.ErrPartnerLinked) {
		t.Fatalf("used invitation was accepted: %v", err)
	}
	if err := database.SetInvitation(t.Context(), "household", "replacement", time.Now().Add(time.Hour)); !errors.Is(err, store.ErrPartnerLinked) {
		t.Fatalf("linked partner could be replaced: %v", err)
	}
}

func TestInvitationClaimPreservesAnExistingFinancialProfile(t *testing.T) {
	database := invitationStore(t, "First", "Saved name")
	h, err := database.Read(t.Context(), "household")
	if err != nil {
		t.Fatal(err)
	}
	h.Members[1].MonthlyTakeHomeCents = 12345
	h.Members[1].DebtCents = 6789
	h, err = database.Update(t.Context(), "household", h)
	if err != nil {
		t.Fatal(err)
	}
	setInvitation(t, database, "hash")
	if err := database.ClaimInvitation(t.Context(), "household", "hash", "two@example.com", "second-subject", "Google name"); err != nil {
		t.Fatal(err)
	}
	saved, err := database.Read(t.Context(), "household")
	if err != nil || !reflect.DeepEqual(saved, h) {
		t.Fatalf("claim replaced a saved financial profile or revision: %v", err)
	}
}

func TestInvitationIdentityConflictsAndProfileFailuresRollBackClaim(t *testing.T) {
	tests := []struct{ name, email, subject, profile string }{
		{"same email", "one@example.com", "other-subject", "Second"},
		{"same subject", "other@example.com", "first-subject", "Second"},
		{"invalid appended profile", "two@example.com", "second-subject", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := invitationStore(t, "First")
			setInvitation(t, database, "hash")
			if err := database.ClaimInvitation(t.Context(), "household", "hash", test.email, test.subject, test.profile); err == nil {
				t.Fatal("invalid identity or profile claimed invitation")
			}
			assertUnchangedPendingClaim(t, database)
			if err := database.ClaimInvitation(t.Context(), "household", "hash", "two@example.com", "second-subject", "Second"); err != nil {
				t.Fatal("rolled-back invitation could not be used", err)
			}
		})
	}
}

func assertUnchangedPendingClaim(t *testing.T, database *store.Store) {
	t.Helper()
	status, err := database.InvitationStatus(t.Context(), "household")
	if err != nil || status.Linked || status.ExpiresAt == nil {
		t.Fatalf("failed claim consumed the invitation or bound identity: %+v %v", status, err)
	}
	h, err := database.Read(t.Context(), "household")
	if err != nil || !reflect.DeepEqual(h, budget.Blank("First")) {
		t.Fatalf("failed claim changed the saved profile: %v", err)
	}
}
