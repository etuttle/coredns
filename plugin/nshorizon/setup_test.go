package nshorizon

import (
	"reflect"
	"strings"
	"testing"

	"github.com/caddyserver/caddy"
)

func TestSetupNsHorizon(t *testing.T) {

	tests := []struct {
		input              string
		shouldErr          bool
		expectedZones      []string
		expectedBackend    string   // expected plugin.
		expectedErrContent string   // substring from the expected error. Empty for positive cases.
	}{
		// positive
		{`nshorizon example.org @kubernetes`, false, []string{"example.org."}, "kubernetes", ""},
		{`nshorizon example.org another.org @kubernetes`, false, []string{"example.org.", "another.org."}, "kubernetes", ""},
		// negative
		{`nshorizon example.org kubernetes`, true, nil, "", "nshorizon: backend field must begin with @"},
		{`nshorizon kubernetes`, true, nil, "", "nshorizon: incomplete config line"},
		{`nshorizon`, true, nil, "", "nshorizon: incomplete config line"},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		nh, be, err := nsHorizonParse(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error but found %s for input %s", i, err, test.input)
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
			}

			if !strings.Contains(err.Error(), test.expectedErrContent) {
				t.Errorf("Test %d: Expected error to contain: %v, found error: %v, input: %s", i, test.expectedErrContent, err, test.input)
			}
		}

		if !test.shouldErr && be != test.expectedBackend {
			t.Errorf("Test %d, Plugin not correctly set for input %s. Expected: %s, actual: %s", i, test.input, test.expectedBackend, be)
		}
		if !test.shouldErr && nh.Zones != nil {
			if !reflect.DeepEqual(test.expectedZones, nh.Zones) {
				t.Errorf("Test %d, wrong zones for input %s. Expected: '%v', actual: '%v'", i, test.input, test.expectedZones, nh.Zones)
			}
		}
	}
}
