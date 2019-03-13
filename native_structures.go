package main

import(
  "strconv"
  "strings"

  "github.com/aws/aws-sdk-go/service/route53"
)

/*
 *  zone container - holds easy to reference data about dns zones
 */
type zone struct {
  domain string
  id string
  recordCount int64
  tld string
}
func (z *zone) DomainToString() string {
  if len(z.tld) > 0 {
    //only print domain and tld if we have a tld
    return z.domain + "." + z.tld
  }

  return z.domain
}
func (z *zone) Serialize() string {
  var jsonString strings.Builder

  //there's got to be a better way...
  jsonString.WriteString("{")
  jsonString.WriteString("\"id\":\"" + z.id + "\",")
  jsonString.WriteString("\"domain\":\"" + z.domain + "\",")
  jsonString.WriteString("\"tld\":\"" + z.tld + "\",")
  jsonString.WriteString("\"recordCount\":" + strconv.FormatInt(z.recordCount, 10))
  jsonString.WriteString("}")

  return jsonString.String()
}


/*
 *  dns record - resource name/values pair
 *  note: values is an array
 *        there is also a reference back to the zone
 */
type record struct {
  name string
  isAlias bool
  zoneRef string
  values []string
}
func (r *record) Serialize() string {
  var jsonString strings.Builder

  jsonString.WriteString("{")
  jsonString.WriteString("\"name\":\"" + r.name + "\",")
  jsonString.WriteString("\"isAlias\":" + strconv.FormatBool(r.isAlias) + ",")
  //don't include if the zoneRef field is blank/empty
  if len(r.zoneRef) > 0 {
    jsonString.WriteString("\"zoneReference\":\"" + r.zoneRef + "\",")
  }

  jsonString.WriteString("\"values\":[\"" + strings.Join(r.values, "\",\"") + "\"]")
  jsonString.WriteString("}")

  return jsonString.String()
}


/*
 *  container for many dns records
 *  map key is the dns record type (resource record type)
 *  map value is an array of record structs
 */
type recordset map[string][]*record
func (rset *recordset) GetDistinctTypes() []string {
  var distinceTypes = make([]string, len(*rset))
  var i int

  for resourceRecordType := range *rset {
    distinceTypes[i] = resourceRecordType
    i++
  }

  return distinceTypes
}
/*
 *  hash api call's response of resource records into a map of dns record types
 */
func (rset *recordset) HashRecordsetTypes(recordsets []*route53.ResourceRecordSet) {
  //for each recordset returned...
  for _, recordset := range recordsets {
    var recordvals strings.Builder
    currentRecordset := new(record)

    //lop off the dot at the end of the recordset name
    currentRecordset.name = string(*recordset.Name)[:len(*recordset.Name)-1]
    if len(recordset.ResourceRecords) > 0 {
      //and parse the resource records (values of the recordset) into a []string
      for j, rval := range recordset.ResourceRecords {
        recordvals.WriteString(*rval.Value)
        if j < len(recordset.ResourceRecords) - 1 {
          recordvals.WriteString(",")
        }
      }
    } else {
      //handle an alias target
      currentRecordset.isAlias = true
      recordvals.WriteString(*recordset.AliasTarget.DNSName)
      currentRecordset.zoneRef = *recordset.AliasTarget.HostedZoneId
    }

    currentRecordset.values = strings.Split(recordvals.String(), ",")
    //add recordset to the correct type bucket
    (*rset)[*recordset.Type] = append((*rset)[*recordset.Type], currentRecordset)
  }
}
func (rset *recordset) SerializeRecords(recordType string) string {
  var specificRecords = (*rset)[strings.ToUpper(recordType)]
  var jsonString strings.Builder

  jsonString.WriteString("{\"zoneRecords\":[")
  for i, rec := range specificRecords {
    jsonString.WriteString(rec.Serialize())
    if i < len(specificRecords) - 1 {
      jsonString.WriteString(",")
    }
  }

  jsonString.WriteString("]}")

  return jsonString.String()
}
