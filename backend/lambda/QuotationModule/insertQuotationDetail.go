package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

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
	//email-SMTP
	from_email = "itsupport@continental.coffee"
	userid     = "itsupport@continental.coffee"
	smtp_pass  = "is@98765"
	// smtp server configuration.
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)


type QuoteDetails struct {
	Update               bool   `json:"update"`
	Quoteid              int    `json:"quoteid"`
	LastQuoteid          int    `json:"last_quoteid"`
	QuoteNumber          string `json:"quote_number"`
	Accountid            string `json:"accountid"`
	Accounttypeid        string `json:"accounttypeid"`
	Contactid            string `json:"contactid"`
	Accounttypename      string `json:"accounttypename"`
	Currencyid           string `json:"currencyid"`
	Currencycode         string `json:"currencycode"`
	Incotermsid          string `json:"incotermsid"`
	Paymentterms         string `json:"payment_terms"`
	BillingId            string `json:"billing_id"`
	Remarksfrommarketing string `json:"remarks_marketing"`
	Destination          string `json:"destination_port"`
	PortLoading          string `json:"port_loading"`
	Otherspecifications  string `json:"other_specification"`
	Destinationcountryid string `json:"destination_countryid"`
	CreatedDate          string `json:"createddate"`
	Createdby            string `json:"createduserid"`
	Finalclientaccountid string `json:"finalclientaccountid"`
	Fromdate             string `json:"fromdate"`
	Todate               string `json:"todate"`
	Modifiedby           string `json:"modifieduserid"`
	Modifieddate         string `json:"modifieddate"`
	Portloadingid        string `json:"portloadingid"`
	Portdestinationid    string `json:"destinationid"`
	Statusid             string `json:"status_id"`
	Masterstatus         string `json:"master_status"`
	PendingWith          string `json:"pending_withuserid"`
	Isactive             bool    `json:"isactive"`
}

type QuoteId struct {
	Id int `json:"quoteid"`
}
func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}


func insertQuotationDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var quote QuoteDetails
	err := json.Unmarshal([]byte(request.Body), &quote)
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
	var rows *sql.Rows

	if quote.Update {

		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
			accountid=$1,
			accounttypeid=$2,
			accounttypename=$3,
			modifieddate=$4,
			modifiedby=$5,
			currencyid=$6,
			currencycode=$7,
			finalaccountid=$8,
			fromdate=$9,
			todate=$10,
			portloadingid=$11,
			destinationid=$12,
			isactive=$13,
			paymentterm=$14,
			otherspecification=$15,
			remarks=$16,
			destinationcountryid=$17,
			destination=$18,
			portloading=$19,
			statusid=$20,
			incotermsid=$21,
			contactid=$22, billing_id=$23, assignedto=$24 where quoteid=$25`

		rows, err = db.Query(sqlStatement1,
			quote.Accountid,
			quote.Accounttypeid,
			quote.Accounttypename,
			quote.Modifieddate,
			quote.Modifiedby,
			quote.Currencyid,
			quote.Currencycode,
			NewNullString(quote.Finalclientaccountid),
			quote.Fromdate,
			quote.Todate,
			quote.Portloadingid,
			NewNullString(quote.Portdestinationid),
			quote.Isactive,
			NewNullString(quote.Paymentterms),
			NewNullString(quote.Otherspecifications),
			NewNullString(quote.Remarksfrommarketing),
			NewNullString(quote.Destinationcountryid),
			NewNullString(quote.Destination),
			quote.PortLoading,
			quote.Statusid,
			quote.Incotermsid,
			quote.Contactid,
			quote.BillingId,
			quote.PendingWith,
			quote.Quoteid)

		if err != nil {
			log.Println("Update to quote table failed")
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_crm_quote_master_newpg where quoteid=$1 order by logid desc limit 1`

		rows1, _ := db.Query(sqlStatementDT1, quote.Quoteid)
		var createdUser, createdDate sql.NullString

		for rows1.Next() {
			err = rows1.Scan(&createdUser, &createdDate)
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for Quotation")
		sqlStatementADT := `INSERT INTO dbo.auditlog_crm_quote_master_newpg(
			quoteid,createdby, created_date, description,modifiedby, modified_date, status)
						VALUES($1,$2,$3,$4,$5,$6,$7)`

		description := "Quotation modified"
		_, errADT := db.Query(sqlStatementADT,
			quote.Quoteid,
			createdUser.String,
			createdDate.String,
			description,
			quote.Modifiedby,
			time.Now().Format("2006-01-02"), quote.Masterstatus)

		log.Println("Audit Update Query Executed")
		if errADT != nil {
			log.Println("unable to update Audit Details", errADT)
		}

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else {

		sqlStatementDT1 := `select quoteid from dbo.crm_quote_master order by quoteid desc limit 1`

		rows1, err := db.Query(sqlStatementDT1)

		// var po InputAdditionalDetails
		for rows1.Next() {
			err = rows1.Scan(&quote.LastQuoteid)
		}

		quote.Quoteid = quote.LastQuoteid + 1
		quote.QuoteNumber = time.Now().Format("20060102") + "-" + strconv.Itoa(quote.Quoteid)

		sqlStatement1 := `INSERT INTO dbo.crm_quote_master (
			accountid,
			accounttypeid,
			accounttypename,
			contactid,
			createddate,
			createdby,
			currencyid,
			currencycode,
			finalaccountid,
			fromdate,
			todate,
			portloadingid,
			destinationid,
			isactive,
			paymentterm,
			otherspecification,
			remarks,
			destinationcountryid,
			destination,
			portloading,
			statusid,
			incotermsid,
			assignedto,
			billing_id,
			quoteid, quotenumber) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25,$26)`

		rows, err = db.Query(sqlStatement1,
			quote.Accountid,
			quote.Accounttypeid,
			quote.Accounttypename,
			quote.Contactid,
			quote.CreatedDate,
			quote.Createdby,
			quote.Currencyid,
			quote.Currencycode,
			NewNullString(quote.Finalclientaccountid),
			quote.Fromdate,
			quote.Todate,
			quote.Portloadingid,
			NewNullString(quote.Portdestinationid),
			quote.Isactive,
			NewNullString(quote.Paymentterms),
			NewNullString(quote.Otherspecifications),
			NewNullString(quote.Remarksfrommarketing),
			NewNullString(quote.Destinationcountryid),
			NewNullString(quote.Destination),
			quote.PortLoading,
			quote.Statusid,
			quote.Incotermsid,
			quote.Createdby, quote.BillingId,
			quote.Quoteid, quote.QuoteNumber)

		if err != nil {
			log.Println("Insert to quote table failed")
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for Quotation")
		sqlStatementADT := `INSERT INTO dbo.auditlog_crm_quote_master_newpg(
										quoteid,createdby, created_date, description, status)
										VALUES($1,$2,$3,$4, $5)`

		description := "Quotation created"
		_, errADT := db.Query(sqlStatementADT,
			quote.Quoteid,
			quote.Createdby,
			time.Now().Format("2006-01-02"),
			description, quote.Masterstatus)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}
		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}
}

func main() {
	lambda.Start(insertQuotationDetail)
}