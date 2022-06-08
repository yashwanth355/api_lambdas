//CHECKEDIN
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type SampleRequestDetails struct {
	AccountId        string     `json:"account_id"`
	AccountName      string     `json:"account_name"`
	ContactId        string     `json:"contact_id"`
	ContactFirstName string     `json:"contact_firstname"`
	ShippingAddress  string     `json:"shipping_address"`
	ShippingId       string     `json:"shipping_id"`
	Remarks          string     `json:"remarks"`
	Masterstatus     string     `json:"masterstatus"`
	RecordType       string     `json:"recordtype"`
	SampleReqDate    string     `json:"samplereq_date"`
	SampleReqNumber  string     `json:"samplereq_number"`
	AuditLogDetails  []AuditLog `json:"audit_log"`
}

type AuditLog struct {
	CreatedDate      string `json:"created_date"`
	CreatedUserName  string `json:"created_username"`
	ModifiedDate     string `json:"modified_date"`
	ModifiedUserName string `json:"modified_username"`
	Description      string `json:"description"`
	Status           string `json:"status"`
}

type Input struct {
	Type     string `json:"type"`
	SampleId int    `json:"sample_id"`
}

func viewSampleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input Input
	err := json.Unmarshal([]byte(request.Body), &input)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")
	var sampleDetails SampleRequestDetails

	var rows *sql.Rows
	log.Println("get Sample Request")
	if input.Type == "viewsample" {
		sqlStatement :=
			`SELECT sr.samplereqno,
			sr.createddate, sr.accountid, acc.accountname, sr.contactid, cont.contactfirstname, concat(s.street,' ', s.city,' ', s.stateprovince, ' ', s.postalcode) as address,sr.shipping_address, sr.remarks, sr.masterstatus
			from dbo.cms_sample_request_master sr
			inner join dbo.accounts_master acc on acc.accountid=sr.accountid
			inner join dbo.contacts_master cont on cont.accountid=sr.accountid
			left join dbo.accounts_shipping_address_master s on s.shippingid=sr.shipping_address
			where sr.samplereqid=$1 limit 1`

		rows, err = db.Query(sqlStatement, input.SampleId)

		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sampleDetails.RecordType = "CCL"
		var remarks, address, shippingId sql.NullString
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&sampleDetails.SampleReqNumber, &sampleDetails.SampleReqDate, &sampleDetails.AccountId, &sampleDetails.AccountName, &sampleDetails.ContactId, &sampleDetails.ContactFirstName,
				&address, &shippingId, &remarks, &sampleDetails.Masterstatus)
			sampleDetails.ShippingAddress = address.String
			sampleDetails.ShippingId = shippingId.String
			sampleDetails.Remarks = remarks.String
		}

		//---------------------Fetch Audit Log Info-------------------------------------//
		log.Println("Fetching Audit Log Info #")
		sqlStatementAI := `select u.username as createduser, a.created_date,
	a.description,a.status, v.username as modifieduser, a.modified_date
   from dbo.auditlog_cms_sample_request_master_newpg a
   inner join dbo.users_master_newpg u on a.createdby=u.userid
   left join dbo.users_master_newpg v on a.modifiedby=v.userid
   where samplereqid=$1 order by logid desc limit 1`
		rowsAI, errAI := db.Query(sqlStatementAI, input.SampleId)
		log.Println("Audit Info Fetch Query Executed")
		if errAI != nil {
			log.Println("Audit Info Fetch Query failed")
			log.Println(errAI.Error())
		}

		var modifiedBy, modifiedDate sql.NullString

		for rowsAI.Next() {
			var al AuditLog
			errAI = rowsAI.Scan(&al.CreatedUserName, &al.CreatedDate, &al.Description, &al.Status, &modifiedBy, &modifiedDate)
			al.ModifiedUserName = modifiedBy.String
			al.ModifiedDate = modifiedDate.String
			auditDetails := append(sampleDetails.AuditLogDetails, al)
			sampleDetails.AuditLogDetails = auditDetails
			log.Println("added one")

		}
		log.Println("Audit Details:", sampleDetails.AuditLogDetails)
	}

	res, _ := json.Marshal(sampleDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(viewSampleRequest)
}
