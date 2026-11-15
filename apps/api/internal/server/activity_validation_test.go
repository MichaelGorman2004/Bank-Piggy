package server

import "testing"

func TestActivityHTTPRejectsInvalidQuery(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	for _, query := range []string{
		"month=2026-13", "month=2026-11&month=2026-12", "month=", "month=9999-01",
		"scope=foreign", "scope=", "scope=member-1&scope=member-2",
		"cursor=not-a-cursor", "cursor=a&cursor=b", "month=%zz", "unknown=value",
	} {
		t.Run(query, func(t *testing.T) {
			requireStatus(t, client.request("GET", "/api/activity?"+query, nil, "", ""), 400)
		})
	}
	body := []byte(`{"expectedPlanRevision":1,"memberId":"member-1"}`)
	requireStatus(t, client.request("POST", "/api/activity/paychecks/foreign/apply?month=2026-11&month=2026-12", body, testOrigin, client.csrf), 400)
}

func TestActivityHTTPRejectsMalformedBodies(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	cases := []struct{ method, path, body string }{
		{"PUT", "/api/connections/foreign/activity", `{}`},
		{"PUT", "/api/connections/foreign/activity", `{"expectedRevision":0,"enabled":true}`},
		{"PUT", "/api/connections/foreign/activity", `{"expectedRevision":0,"enabled":true,"accounts":null}`},
		{"POST", "/api/activity/paychecks/foreign/apply", `{"memberId":"member-1"}`},
		{"POST", "/api/activity/paychecks/foreign/apply", `{"expectedPlanRevision":1,"memberId":null}`},
		{"POST", "/api/activity/paychecks/foreign/apply", `null`},
	}
	for _, item := range cases {
		requireStatus(t, client.request(item.method, item.path, []byte(item.body), testOrigin, client.csrf), 400)
	}
}

func TestActivityHTTPRemovesIndividualRecordRoute(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	body := []byte(`{"expectedRevision":1,"category":"spending"}`)
	requireStatus(t, client.request("PUT", "/api/activity/records/record", body, testOrigin, client.csrf), 404)
	requireStatus(t, client.request("GET", "/api/activity?cursor=", nil, "", ""), 400)
}
