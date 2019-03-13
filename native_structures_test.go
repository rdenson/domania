package main
import (
  "encoding/json"
  "testing"
)

/*
 *  helper function for mocking the "recordset" type
 */
func createRecordset() recordset {
  r0 := &record{
    name: "A",
    values: []string{"127.0.0.1"},
  }
  r1 := &record{
    name: "AAAA",
    values: []string{"::1"},
  }
  r2 := &record{
    name: "CNAME",
    values: []string{"howdy.io"},
  }
  r3 := &record{
    name: "CNAME",
    values: []string{"roundup.net"},
  }
  zoneRecordset := make(recordset)

  zoneRecordset["A"] = append(zoneRecordset["A"], r0)
  zoneRecordset["AAAA"] = append(zoneRecordset["AAAA"], r1)
  zoneRecordset["CNAME"] = append(zoneRecordset["CNAME"], r2, r3)

  return zoneRecordset
}

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

  if !json.Valid([]byte(tc1)) {
    t.Errorf("tc1 - invalid JSON: %s", tc1)
  }
}

func TestRecordsetSerializeRecords(t *testing.T) {
  var jsonObject map[string][]*record
  var zr recordset = createRecordset()

  //test cases
  //  1: serialize function returns valid json
  //  2: key: "zoneRecords" exist and has the expected length
  serialized := zr.SerializeRecords("cname")
  json.Unmarshal([]byte(serialized), &jsonObject)

  if !json.Valid([]byte(serialized)) {
    t.Errorf("tc1 - invalid JSON: %s", serialized)
  }

  if jsonObject["zoneRecords"] == nil || len(jsonObject["zoneRecords"]) != 2 {
    t.Errorf("tc2 - expected key, \"zoneRecords\" is missing or has the wrong number of elements: %+v", jsonObject)
  }
}
