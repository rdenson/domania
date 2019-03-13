package main
import (
  "encoding/json"
  "testing"
)

func TestZoneDomainToString(t *testing.T) {
  zoneWithTld := &zone{
    domain: "somedomain",
    tld: "org",
  }
  zoneWithTldExpectedValue := "somedomain.org"
  zoneWithoutTld := &zone{
    domain: "test-domain",
  }
  zoneWithoutTldExpectedValue := "test-domain"

  tc1 := zoneWithTld.DomainToString()
  tc2 := zoneWithoutTld.DomainToString()

  if tc1 != zoneWithTldExpectedValue {
    t.Errorf("tc1 - expected \"%s\" but found: \"%s\"", zoneWithTldExpectedValue, tc1)
  }

  if tc2 != zoneWithoutTldExpectedValue {
    t.Errorf("tc2 - expected \"%s\" but found: \"%s\"", zoneWithoutTldExpectedValue, tc2)
  }
}

func TestZoneSerialize(t *testing.T) {
  z := new(zone)
  tc1 := z.Serialize()

  if !json.Valid([]byte(tc1)) {
    t.Errorf("tc1 - invalid JSON: %s", tc1)
  }
}

func TestRecordSerialize(t *testing.T) {
  r := new(record)
  tc1 := r.Serialize()

  if !json.Valid([]byte(tc1)){
    t.Errorf("tc1 - invalid JSON: %s", tc1)
  }
}
