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

	//SES
	"bytes"
	
	// "text/template"
	//go get -u github.com/aws/aws-sdk-go
	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	// "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"gopkg.in/gomail.v2"
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

type Email struct {
	ToEmail string `json:"to_email"`
	ToName  string `json:"name"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type LeadDetails struct {
	Update                     bool           `json:"update"`
	LeadId                     string         `json:"leadid"`
	LastIdsno                  int            `json:"lastidsno"`
	Idsno                      int            `json:"idsno"`
	Accountname                string         `json:"accountname"`
	Aliases                    string         `json:"aliases"`
	Accounttypeid              string         `json:"accounttypeid"`
	Website                    string         `json:"website"`
	Approximativeannualrevenue string         `json:"approxannualrev"`
	Productsegmentid           string         `json:"productsegmentid"`
	ContactSalutationid        int            `json:"contact_salutationid"`
	Contactfirstname           string         `json:"contact_firstname"`
	Contactlastname            string         `json:"contact_lastname"`
	ContactPosition            string         `json:"contact_position"`
	ContactEmail               string         `json:"contact_email"`
	ContactPhone               string         `json:"contact_phone"`
	ContactExtId               string         `json:"contact_ext"`
	ContactMobile              string         `json:"contact_mobile"`
	Manfacunit                 bool            `json:"manfacunit"`
	Instcoffee                 bool            `json:"instcoffee"`
	SampleReady                bool            `json:"sample_ready"`
	Coffeetypeid               string         `json:"coffeetypeid"`
	OtherInformation           string         `json:"otherinformation"`
	BillingStreetAddress       string         `json:"billing_street"`
	BillingCity                string         `json:"billing_citycode"`
	BillingState               string         `json:"billing_statecode"`
	BillingPostalCode          string         `json:"billing_postalcode"`
	BillingCountry             string         `json:"billing_countrycode"`
	ContactStreetAddress       string         `json:"contact_street"`
	ContactCity                string         `json:"contact_citycode"`
	ContactState               string         `json:"contact_statecode"`
	ContactPostalCode          string         `json:"contact_postalcode"`
	ContactCountry             string         `json:"contact_countrycode"`
	CreatedDate                string         `json:"createddate"`
	CreatedUserid              string         `json:"createduserid"`
	LdCreatedUserid            string         `json:"ldcreateduserid"`
	LdCreatedUserName          string         `json:"ldcreatedusername"`
	UserEmail                  string         `json:"emailid"`
	ModifiedDate               string         `json:"modifieddate"`
	ModifiedUserid             string         `json:"modifieduserid"`
	ShippingContinentid        string         `json:"shipping_continentid"`
	ShippingCountryid          string         `json:"countryid"`
	Leadscore                  int            `json:"leadscore"`
	Masterstatus               string         `json:"masterstatus"`
	// Approvalstatus             bool            `json:"approvalstatus"`
	ShippingContinent          string         `json:"shipping_continent"`
	ShippingCountry            string         `json:"shipping_country"`
	Isactive                   bool            `json:"isactive"`
	AuditLogDetails            []AuditLogGCPO `json:"audit_log_gc_po"`
}

type LeadId struct {
	Id string `json:"leadid"`
}
type AuditLogGCPO struct {
	CreatedDate    string `json:"createddate"`
	CreatedUserid  string `json:"createduserid"`
	ModifiedDate   string `json:"modifieddate"`
	ModifiedUserid string `json:"modifieduserid"`
	Description    string `json:"description"`
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
type Recipient struct {
	toEmails  []string
	ccEmails  []string
	bccEmails []string
}
func insertLeadDetails(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var lead LeadDetails
	var email Email
	var audit AuditLogGCPO
	err := json.Unmarshal([]byte(request.Body), &lead)
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
	ctx = context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}


	// var rows *sql.Rows
	// Find created user username
	sqlStatementFUser1 := `SELECT username 
						FROM dbo.users_master_newpg where userid=$1`
	row := tx.QueryRow(sqlStatementFUser1, lead.CreatedUserid)
	err = row.Scan(&lead.LdCreatedUserName)
	
	if err != nil {
		tx.Rollback()
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		
	}
	//Find Duplicate Lead:
	log.Println("Finding Duplicate Leadname in DB")
	var duplicatelead int
	sqlStatementDupLead := `select count(accountname) from dbo.cms_leads_master where accountname like $1`
	row = tx.QueryRow(sqlStatementDupLead, lead.Accountname)
	err = row.Scan(&duplicatelead)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	

	if lead.Update {
		log.Println("Entered Update Leads Segment")
		log.Println(lead.Productsegmentid)

		sqlStatementU1 := `UPDATE dbo.cms_leads_master SET 
							accountname=$1,
							accounttypeid=$2,
							contact_mobile=$3,
							email=$4,
							phone=$5,
							modifieddate=$6,
							modifieduserid=$7,
							shipping_continentid=$8,
							countryid=$9,
							approxannualrev=$10,
							website=$11,
							productsegmentid=$12,
							leadscore=$13,
							contactfirstname=$14,
							contactlastname=$15,
							manfacunit=$16,
							instcoffee=$17,
							sampleready=$18,
							contact_salutationid=$19,
							contact_position=$20,
							shipping_continent=$21,
							shipping_country=$22,
							coffeetypeid=$23,
							aliases=$24,
							otherinformation=$25,
							contact_ext_id=$26
							where leadid=$27`

			_, err = tx.ExecContext(ctx,sqlStatementU1,
			lead.Accountname,
			lead.Accounttypeid,
			lead.ContactMobile,
			lead.ContactEmail,
			lead.ContactPhone,
			lead.ModifiedDate,
			lead.ModifiedUserid,
			lead.ShippingContinentid,
			lead.ShippingCountryid,
			NewNullString(lead.Approximativeannualrevenue),
			lead.Website,
			lead.Productsegmentid,
			lead.Leadscore,
			lead.Contactfirstname,
			lead.Contactlastname,
			lead.Manfacunit,
			lead.Instcoffee,
			lead.SampleReady,
			lead.ContactSalutationid,
			lead.ContactPosition,
			lead.ShippingContinent,
			lead.ShippingCountry,
			lead.Coffeetypeid,
			lead.Aliases,
			lead.OtherInformation,
			lead.ContactExtId,
			lead.LeadId)
		log.Println("Update lead successful")
		if err != nil {
			tx.Rollback()
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatement3 := `UPDATE dbo.cms_leads_billing_address_master SET street=$1, city=$2, stateprovince=$3, postalcode=$4, country=$5 where billingid=$6`

		_, err = tx.ExecContext(ctx,sqlStatement3, lead.BillingStreetAddress, lead.BillingCity, lead.BillingState, lead.BillingPostalCode, lead.BillingCountry, lead.LeadId)
		if err != nil {
			tx.Rollback()
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		sqlStatement4 := `UPDATE dbo.cms_leads_shipping_address_master SET street=$1, city=$2, stateprovince=$3, postalcode=$4, country=$5 where shippingid=$6`

		_, err = tx.ExecContext(ctx,sqlStatement4, lead.ContactStreetAddress, lead.ContactCity, lead.ContactState, lead.ContactPostalCode, lead.ContactCountry, lead.LeadId)

		if err != nil {
			tx.Rollback()
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		// Insert Audit Info
		log.Println("Entered Audit Module for PO Type")
		// Find created user username
		sqlStatementAUser1 := `SELECT u.userid,u.username 
								FROM dbo.users_master_newpg u
								inner join 
								dbo.auditlog_cms_leads_master_newpg ld 
								on ld.createdby=u.userid
								where ld.leadid=$1`
		row = tx.QueryRow(sqlStatementAUser1, lead.LeadId)
		err = row.Scan(&lead.LdCreatedUserid, &lead.LdCreatedUserName)
		if err != nil {
			tx.Rollback()
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		audit.Description = "Lead Details Modified"
		// sd.InvoiceDate = time.Now().Format("2006-01-02")
		audit.ModifiedDate = time.Now().Format("2006-01-02")
		audit.ModifiedUserid = lead.ModifiedUserid

		sqlStatementADT := `update
							dbo.auditlog_cms_leads_master_newpg
							set
							description=$1,
							modifiedby=$2,
							modified_date=$3
							where
							leadid=$4`
		_, err = tx.ExecContext(ctx,sqlStatementADT,
			audit.Description,
			audit.ModifiedUserid,
			audit.ModifiedDate,
			lead.LeadId)

		log.Println("Audit Insert Query Executed")
		if err != nil {
			tx.Rollback()
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		//Insert Notification
		sqlStatementNotif := `insert into dbo.notifications_master_newpg(userid,objid,feature_category,status) 
							values($1,$2,'Lead','Lead Updated')`
		_, err = tx.ExecContext(ctx,sqlStatementNotif, lead.ModifiedUserid, lead.LeadId)
		if err != nil {
			tx.Rollback()
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		res, _ := json.Marshal(row)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else if duplicatelead == 0 {
		log.Println("Entered Lead creation segment")
		log.Println(lead.Productsegmentid)
		//Find latest poid
		sqlStatementLD1 := `SELECT idsno 
							FROM dbo.cms_leads_master 
							where idsno is not null
							ORDER BY idsno DESC 
							LIMIT 1`
		row = tx.QueryRow(sqlStatementLD1)
		err = row.Scan(&lead.LastIdsno)
		if err != nil {
			tx.Rollback()
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		//Generating PO NOs----------------
		lead.Idsno = lead.LastIdsno + 1
		lead.LeadId = "Lead-" + strconv.Itoa(lead.Idsno)

		sqlStatement1 := `INSERT INTO dbo.cms_leads_master (
			leadid,
			autogencode,
			legacyid,
			accountname,
			accounttypeid,
			phone,
			email,
			createddate,
			createduserid,
			shipping_continentid,
			countryid,
			approxannualrev,
			website,
			productsegmentid,
			leadscore,
			masterstatus,
			contactfirstname,
			contactlastname,
			manfacunit,
			instcoffee,
			sampleready,
			contact_salutationid,
			contact_position,
			contact_mobile,
			shipping_continent,
			shipping_country,
			coffeetypeid,
			aliases,
			isactive,
			otherinformation,
			contact_ext_id) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28,$29,$30,$31)`

		_, err = tx.ExecContext(ctx,sqlStatement1,
			lead.LeadId,
			lead.LeadId,
			lead.Idsno,
			lead.Accountname,
			lead.Accounttypeid,
			lead.ContactPhone,
			lead.ContactEmail,
			lead.CreatedDate,
			lead.CreatedUserid,
			lead.ShippingContinentid,
			lead.ShippingCountryid,
			NewNullString(lead.Approximativeannualrevenue),
			lead.Website,
			lead.Productsegmentid,
			lead.Leadscore,
			lead.Masterstatus,
			lead.Contactfirstname,
			lead.Contactlastname,
			lead.Manfacunit,
			lead.Instcoffee,
			lead.SampleReady,
			lead.ContactSalutationid,
			lead.ContactPosition,
			lead.ContactMobile,
			lead.ShippingContinent,
			lead.ShippingCountry,
			lead.Coffeetypeid,
			lead.Aliases,
			lead.Isactive,
			lead.OtherInformation,
			lead.ContactExtId)

		
		if err != nil {
			tx.Rollback()
			log.Println("Insert to lead table failed")
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		//Insert Notification
		sqlStatementNotif := `insert into dbo.notifications_master_newpg(userid,objid,feature_category,status) 
							values($1,$2,'Lead','Lead Created')`
		_, _ = db.Query(sqlStatementNotif, lead.CreatedUserid, lead.LeadId)

		sqlStatement3 := `INSERT INTO dbo.cms_leads_billing_address_master(
							leadid, billingid, street, city, stateprovince, postalcode, country) VALUES ($1, $2, $3, $4, $5, $6, $7)`

		_, err = tx.ExecContext(ctx,sqlStatement3, lead.LeadId, lead.LeadId, lead.BillingStreetAddress, lead.BillingCity, lead.BillingState, lead.BillingPostalCode, lead.BillingCountry)
		if err != nil {
			tx.Rollback()
			
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		log.Println("Insert into Leads Billing table")
		sqlStatement4 := `INSERT INTO dbo.cms_leads_shipping_address_master (leadid, shippingid, street, city, stateprovince, postalcode, country) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err = tx.ExecContext(ctx,sqlStatement4, lead.LeadId, lead.LeadId, lead.ContactStreetAddress, lead.ContactCity, lead.ContactState, lead.ContactPostalCode, lead.ContactCountry)
		if err != nil {
			tx.Rollback()			
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		log.Println("Insert into Leads Shipping table")
		
		email.Message = "New lead has been created"
		email.ToEmail = lead.UserEmail
		// triggerSESEmail(email.ToEmail,email.Message,lead.Accountname,lead.ShippingCountry,lead.LdCreatedUserName)

		message := email.Message
		subject := email.Message
		fromEmail := "itsupport@continental.coffee"

		recipient := Recipient{
			toEmails:  []string{"sriram.n@continental.coffee"},
			ccEmails:  []string{""},
			bccEmails: []string{""},
		}

		attachments := []string{"https://s3.ap-south-1.amazonaws.com/dev.cclproducts.com/dev.cclproducts.com/Debit_Note_Release_CCL_Accounts.pdf"}

		SendEmailRawSES(message, subject, fromEmail, recipient, attachments)
		// //------------------Insert Audit Info----------------------------
		log.Println("Entered Audit Module for Lead Module")
		audit.CreatedUserid = lead.CreatedUserid
		audit.CreatedDate = lead.CreatedDate
		audit.Description = "Lead Created"
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_leads_master_newpg(
						leadid,createdby, created_date, description)
						VALUES($1,$2,$3,$4)`
		_, errADT := db.Query(sqlStatementADT,
			lead.LeadId,
			audit.CreatedUserid,
			audit.CreatedDate,
			audit.Description)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}

		// res, _ := json.Marshal(value)
		log.Println("Commit the transaction.")
		 if err = tx.Commit(); err != nil {
			log.Println("Error while committing the transaction")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		
		}
		return events.APIGatewayProxyResponse{200, headers, nil, string("Lead saved successfully"), false}, nil

	} else if duplicatelead > 0{
		log.Println("Lead Name already exists")
		return events.APIGatewayProxyResponse{230, headers, nil, "Lead Name already exists", false}, nil
	}
	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}
func main() {
	lambda.Start(insertLeadDetails)
}
// func triggerSESEmail(to_email,message,account_name,account_country,account_owner string) (string, error) {
	
	
// 	data := "{ \"MessageToUser\":\"" + message + "\", \"AccountName\": \"" + account_name + "\",\"AccountCountry\": \"" + account_country + "\",\"AccountOwner\": \"" + account_owner + "\"}"

// 	// Create a new session in the us-west-2 region.
// 	// Replace us-west-2 with the AWS Region you're using for Amazon SES.
// 	sess, err := session.NewSession(&aws.Config{
// 		Region:      aws.String("ap-south-1"),
		
		
// 		Credentials: credentials.NewStaticCredentials("AKIAW4SF47I56UGUFEG5", "pOUhTTXsawxywBClrLJk+7a4flIqOupQMZIkgqxN", ""),
// 	})

// 	// Create an SES session.
// 	svc := ses.New(sess)

// 	template := "EmailOnLeadCreation"
// 	from := "itsupport@continental.coffee"
// 	to := "sriram.n@continental.coffee"
// 	filename := "https://s3.ap-south-1.amazonaws.com/dev.cclproducts.com/dev.cclproducts.com/Debit_Note_Release_CCL_Accounts.pdf"
// 	input := &ses.SendTemplatedEmailInput{
// 		Source:   &from,
// 		Template: &template,
// 		Destination: &ses.Destination{
// 			ToAddresses: []*string{&to},
// 		},
// 		TemplateData: &data,
// 		Attach: &filename,
				
// 	}
// 	result, err := svc.SendTemplatedEmail(input)

// 	// Display error messages if they occur.
// 	if err != nil {
// 		if aerr, ok := err.(awserr.Error); ok {
// 			switch aerr.Code() {
// 			case ses.ErrCodeMessageRejected:
// 				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
// 			case ses.ErrCodeMailFromDomainNotVerifiedException:
// 				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
// 			case ses.ErrCodeConfigurationSetDoesNotExistException:
// 				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
// 			default:
// 				fmt.Println(aerr.Error())
// 			}
// 		} else {
// 			// Print the error, cast err to awserr.Error to get the Code and
// 			// Message from an error.
// 			fmt.Println(err.Error())
// 		}

// 		return "Success", nil
// 	}

// 	fmt.Println("Email Sent to address: " + to)
// 	fmt.Println(result)
// 	return "Success", nil
// }
func SendEmailRawSES(messageBody string, subject string, fromEmail string, recipient Recipient, attachments []string) {

	// create new AWS session
	sess, err := session.NewSession(&aws.Config{
					Region:      aws.String("ap-south-1")},
				)
	if err != nil {
		log.Println("Error occurred while creating aws session", err)
		return
	}

	// create raw message
	msg := gomail.NewMessage()

	// set to section
	var recipients []*string
	for _, r := range recipient.toEmails {
		recipient := r
		recipients = append(recipients, &recipient)
	}

	// Set to emails
	msg.SetHeader("To", recipient.toEmails...)

	// cc mails mentioned
	if len(recipient.ccEmails) != 0 {
		// Need to add cc mail IDs also in recipient list
		for _, r := range recipient.ccEmails {
			recipient := r
			recipients = append(recipients, &recipient)
		}
		msg.SetHeader("cc", recipient.ccEmails...)
	}

	// bcc mails mentioned
	if len(recipient.bccEmails) != 0 {
		// Need to add bcc mail IDs also in recipient list
		for _, r := range recipient.bccEmails {
			recipient := r
			recipients = append(recipients, &recipient)
		}
		msg.SetHeader("bcc", recipient.bccEmails...)
	}

	// create an SES session.
	svc := ses.New(sess)

	msg.SetAddressHeader("From", fromEmail, "<name>")
	msg.SetHeader("To", recipient.toEmails...)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", messageBody)

	// If attachments exists
	if len(attachments) != 0 {
		for _, f := range attachments {
			msg.Attach(f)
		}
	}

	// create a new buffer to add raw data
	var emailRaw bytes.Buffer
	msg.WriteTo(&emailRaw)

	// create new raw message
	message := ses.RawMessage{Data: emailRaw.Bytes()}

	input := &ses.SendRawEmailInput{Source: &fromEmail, Destinations: recipients, RawMessage: &message}

	// send raw email
	_, err = svc.SendRawEmail(input)
	if err != nil {
		log.Println("Error sending mail - ", err)
		return
	}

	log.Println("Email sent successfully to: ", recipient.toEmails)
}