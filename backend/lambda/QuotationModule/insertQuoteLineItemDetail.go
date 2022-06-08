package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"strconv"
	"text/template"
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
	smtp_pass  = "is@98765"
	// smtp server configuration.
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)

const packagingDepTemp = `<!DOCTYPE html>
	    <html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		</head>
		<body>
		    <p>Hi Team,</p>
			<p>{{.EMessage}}</p>
			<p>Note :This is an auto-generated email please don't reply to this email. In case of any queries reach out to shilpa.a@continental.coffee.</p>
			<p>Regards,</p>
			<p>a2zcclproducts</p>
		</body>
	</html>`
const marketingExecutiveTemp = `<!DOCTYPE html>
		<html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		</head>
		<body>
			<p>Hi Team,</p>
			<p>{{.EMessage}}</p>
			<p>Note :This is an auto-generated email please don't reply to this email. In case of any queries reach out to shireesh.gs@continental.coffee</p>
			<p>Regards,</p>
			<p>a2zcclproducts</p>
		</body>
		</html>`

// {{.VEmail}}

// Email is input request data which is used for sending email using aws ses service
type Email struct {
	ToEmail string `json:"to_email"`
	ToName  string `json:"name"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type QuoteItemDetails struct {
	Update                 bool   `json:"update"`
	QuoteLineId            int    `json:"quotelineitem_id"`
	LastQuoteLineId        int    `json:"lastquotelineitem_id"`
	QuoteLineNumber        string `json:"quoteline_number"`
	QuoteId                int    `json:"quote_id"`
	SampleId               string `json:"sample_id"`
	PackingId              int    `json:"category_id"`
	PackingTypeId          int    `json:"categorytype_id"`
	WeightId               int    `json:"weight_id"`
	CartonTypeId           int    `json:"cartontype_id"`
	CapTypeId              int    `json:"captype_id"`
	SecondaryId            int    `json:"secondary_id"`
	NoOfSecondaryId        int    `json:"noofsecondary_id"`
	UPCId                  int    `json:"upc_id"`
	Palletizationrequireid int    `json:"palletizationrequire_id"`
	CustomerBrandName      string `json:"customerbrand_name"`
	AdditionalRequirements string `json:"additional_req"`
	ExpectedOrder          int    `json:"expectedorder_kgs"`
	CreatedDate            string `json:"created_date"`
	CreatedBy              string `json:"created_by"`
	CreatedByUserId        string `json:"created_byuserid"`
	Modifieddate           string `json:"modified_date"`
	Modifiedby             string `json:"modified_by"`
	ModifiedbyUserId       string `json:"modified_byuserid"`
	MasterStatus           string `json:"master_status"`

	IsReqNewPacking       int    `json:"isreqnew_packing"`
	NewPackingDescription string `json:"taskdesc"`
}

func insertQuoteLineItemDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var quoteItem QuoteItemDetails
	var email Email
	err := json.Unmarshal([]byte(request.Body), &quoteItem)
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

	if quoteItem.Update {

		sqlStatement1 := `UPDATE dbo.cms_quote_item_master SET 
							quoteid=$1,
							sampleid=$2,
							packcategoryid=$3,
							packcategorytypeid=$4,
							packweightid=$5,
							packcartontypeid=$6,
							packcaptypeid=$7,
							packsecondaryid=$8,
							packnoofsecondaryid=$9,
							packupcid=$10,
							palletizationrequireid=$11,
							expectedorderqty=$12,
							customerbrandname=$13,
							additionalrequirements=$14,
							is_req_new_pack=$15,
							modifieddate=$16,
							modifiedby=$17 where quoteitemid=$18`

		rows, err = db.Query(sqlStatement1,
			quoteItem.QuoteId,
			quoteItem.SampleId,
			quoteItem.PackingId,
			quoteItem.PackingTypeId,
			quoteItem.WeightId,
			quoteItem.CapTypeId,
			quoteItem.CapTypeId,
			quoteItem.SecondaryId,
			quoteItem.NoOfSecondaryId,
			quoteItem.UPCId,
			quoteItem.Palletizationrequireid,
			quoteItem.ExpectedOrder,
			quoteItem.CustomerBrandName,
			quoteItem.AdditionalRequirements,
			quoteItem.IsReqNewPacking,
			quoteItem.Modifieddate,
			quoteItem.Modifiedby,
			quoteItem.QuoteLineId)

		if err != nil {
			log.Println("Insert to quote line table failed")
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_cms_quote_item_master_newpg where quoteitemid=$1 order by logid desc limit 1`

		rows1, _ := db.Query(sqlStatementDT1, quoteItem.QuoteLineId)

		for rows1.Next() {
			err = rows1.Scan(&quoteItem.CreatedByUserId, &quoteItem.CreatedDate)
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for Quotation line item")
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_quote_item_master_newpg(
			quoteitemid,createdby, created_date, description,modifiedby, modified_date, status)
						VALUES($1,$2,$3,$4,$5,$6,$7)`

		description := "Quotation line item modified"
		_, errADT := db.Query(sqlStatementADT,
			quoteItem.QuoteLineId,
			quoteItem.CreatedByUserId,
			quoteItem.CreatedDate,
			description,
			quoteItem.ModifiedbyUserId,
			time.Now().Format("2006-01-02"), quoteItem.MasterStatus)

		log.Println("Audit Update Query Executed")
		if errADT != nil {
			log.Println("unable to update Audit Details", errADT)
		}
	} else {

		sqlStatementDT1 := `select quoteitemid from dbo.cms_quote_item_master order by quoteitemid desc limit 1`

		rows1, err := db.Query(sqlStatementDT1)

		for rows1.Next() {
			err = rows1.Scan(&quoteItem.LastQuoteLineId)
		}

		sqlStatementDT2 := `select q.quotenumber, a.accountname, a.createduserid from dbo.crm_quote_master q
		Inner join dbo.accounts_master a on a.accountid= q.accountid
		 where quoteid=$1`
		rows2, err := db.Query(sqlStatementDT2, quoteItem.QuoteId)

		var quoteNumber, accountName, accountCreatedUserId, marketingExecutiveEmail sql.NullString
		for rows2.Next() {
			err = rows2.Scan(&quoteNumber, &accountName, &accountCreatedUserId)
		}

		sqlStatementDT3 := `select emailid from dbo.users_master_newpg where userid=$1`

		rows3, err := db.Query(sqlStatementDT3, quoteItem.CreatedByUserId)

		for rows3.Next() {
			err = rows3.Scan(&marketingExecutiveEmail)
		}

		quoteItem.QuoteLineId = quoteItem.LastQuoteLineId + 1
		quoteItem.QuoteLineNumber = time.Now().Format("20060102") + "-" + strconv.Itoa(quoteItem.QuoteLineId)

		if quoteItem.IsReqNewPacking == 1 {

			var lastTaskIdsNo, newTaskIdsNo int
			sqlStatementDT2 := `select taskidsno from dbo.project_project_management_tasks_master order by taskidsno desc limit 1`

			rows2, _ := db.Query(sqlStatementDT2)

			for rows2.Next() {
				err = rows2.Scan(&lastTaskIdsNo)
			}

			newTaskIdsNo = lastTaskIdsNo + 1
			taskId := "FAC-" + strconv.Itoa(newTaskIdsNo)

			sqlStatement5 := `INSERT INTO dbo.project_project_management_tasks_master (
				taskid,
				taskidsno,
				projectid,
				taskname,
				taskstart,
				taskdesc,
				closestatus,
				cancelstatus,
				assignedto,
				status,
				sdate,taskstartdatetime, custid, originid) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

			_, err := db.Query(sqlStatement5,
				taskId,
				newTaskIdsNo,
				"FAC-3",
				"SALES DEPARTMENT HAS REQUESTED FOR THE PACKING TYPE",
				time.Now().Format("2006-01-02"),
				quoteItem.NewPackingDescription, false, false, "1634183543447", "Not Started", time.Now().Format("2006-01-02"), time.Now(), accountCreatedUserId.String, quoteItem.QuoteLineNumber)

			if err != nil {
				log.Println("Insert to task table failed", err.Error())
			}

			sqlStatement1 := `INSERT INTO dbo.cms_quote_item_master (
				quoteid,
				sampleid,
				expectedorderqty,
				customerbrandname,
				is_req_new_pack,
				new_pack_task_status,
				new_pack_desc,
				packingtaskid,
				createddate,
				createdby,quoteitemid, lineitemnumber) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

			rows, err = db.Query(sqlStatement1,
				quoteItem.QuoteId,
				quoteItem.SampleId,
				quoteItem.ExpectedOrder,
				quoteItem.CustomerBrandName,
				quoteItem.IsReqNewPacking,
				"New Packing Type Requested",
				quoteItem.NewPackingDescription,
				taskId,
				quoteItem.CreatedDate,
				quoteItem.CreatedBy, quoteItem.QuoteLineId, quoteItem.QuoteLineNumber)

			// send email to marketing exceutive
			email.ToEmail = marketingExecutiveEmail.String
			sub2 := "Quotation: " + quoteNumber.String + " New PackType Request Task # " + taskId + " - Action required."
			email.Message = "New packing type " + quoteItem.NewPackingDescription + " is created for / Quotation: " + quoteNumber.String + " with Task # " + taskId + "."
			smtpSendEmail(packagingDepTemp, sub2, email.Message, email.ToEmail)

			// send email to packaging department head
			email.ToEmail = "kalyan@kasvibes.com"
			sub := "Customer: " + accountName.String + " /Quotation: " + quoteNumber.String + " New Pack Type Request Task # " + taskId + " -Action Required"
			email.Message = "The New pack Type Task # " + taskId + " for customer: " + accountName.String + " /Quotation: " + quoteNumber.String + " is assigned to you."
			smtpSendEmail(packagingDepTemp, sub, email.Message, email.ToEmail)

		} else {
			sqlStatement1 := `INSERT INTO dbo.cms_quote_item_master (
			quoteid,
			sampleid,
			packcategoryid,
			packcategorytypeid,
			packweightid,
			packcartontypeid,
			packcaptypeid,
			packsecondaryid,
			packnoofsecondaryid,
			packupcid,
			palletizationrequireid,
			expectedorderqty,
			customerbrandname,
			additionalrequirements,
			is_req_new_pack,
			createddate,
			createdby,quoteitemid, lineitemnumber) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

			rows, err = db.Query(sqlStatement1,
				quoteItem.QuoteId,
				quoteItem.SampleId,
				quoteItem.PackingId,
				quoteItem.PackingTypeId,
				quoteItem.WeightId,
				quoteItem.CapTypeId,
				quoteItem.CapTypeId,
				quoteItem.SecondaryId,
				quoteItem.NoOfSecondaryId,
				quoteItem.UPCId,
				quoteItem.Palletizationrequireid,
				quoteItem.ExpectedOrder,
				quoteItem.CustomerBrandName,
				quoteItem.AdditionalRequirements,
				quoteItem.IsReqNewPacking,
				quoteItem.CreatedDate,
				quoteItem.CreatedBy, quoteItem.QuoteLineId, quoteItem.QuoteLineNumber)
		}

		if err != nil {
			log.Println("Insert to quote line table failed")
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for Quotation line item")
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_quote_item_master_newpg(
										quoteitemid,createdby, created_date, description, status)
										VALUES($1,$2,$3,$4, $5)`

		description := "Quotation line item created"
		_, errADT := db.Query(sqlStatementADT,
			quoteItem.QuoteLineId,
			quoteItem.CreatedByUserId,
			time.Now().Format("2006-01-02"),
			description, quoteItem.MasterStatus)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}
	}

	res, _ := json.Marshal(rows)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(insertQuoteLineItemDetail)
}

func smtpSendEmail(temp, subject, message, to_email string) (string, error) {
	log.Println("Entered SMTP Email Module")
	// Receiver email address.
	to := []string{
		to_email,
	}
	// Authentication.
	auth := smtp.PlainAuth("", from_email, smtp_pass, smtpHost)

	t := template.Must(template.New(temp).Parse(temp))

	var body bytes.Buffer

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject:"+subject+"\n%s\n\n", mimeHeaders)))

	t.Execute(&body, struct {
		EMessage string
	}{
		EMessage: message,
	})

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to, body.Bytes())
	if err != nil {
		fmt.Println(err)

	}
	return "Email Sent!", nil
}
