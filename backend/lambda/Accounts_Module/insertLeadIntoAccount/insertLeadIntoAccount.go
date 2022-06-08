package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"

	//SES
	// "bytes"
	// "net/smtp"
	// "text/template"
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
const generalTemp = `<!DOCTYPE html>
	    <html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		<style>
		table {
  		font-family: arial, sans-serif;
  		border-collapse: collapse;
  		width: 100%;
		}
		td, th {
  		border: 1px solid #dddddd;
  		text-align: left;
  		padding: 8px;
		}
		tr:nth-child(even) {
  		background-color: #dddddd;
		}
		</style>
		</head>
		<body>
		<h3>Hi,</h3>
			<p>{{.EMessage}}</p>
		
		<p>Regards,</p>
		<p>a2z.cclproducts</p>
		</body>
		</html>`

type Email struct {
	ToEmail string `json:"to_email"`
	ToName  string `json:"name"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type AccountDetails struct {
	LastAccIdsno         int    `json:"lastaccidsno"`
	AccIdsno             int    `json:"accidsno"`
	LeadId               string `json:"leadid"`
	AccountId            string `json:"account_id"`
	Role                 string `json:"role"`
	ConvertLeadToAccount bool   `json:"convertleadtoaccount"`
	Approve              bool   `json:"approve"`
	Reject               bool   `json:"reject"`
	Comments             string `json:"comments"`
	UserEmail            string `json:"emailid"`
	UserName             string `json:"username"`
}

var psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
var rows *sql.Rows

func insertLeadIntoAccount(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		// return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	var lead AccountDetails
	// var email Email
	err = json.Unmarshal([]byte(request.Body), &lead)

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")

	if (lead.Role == "Marketing Executive") && (lead.ConvertLeadToAccount) {

		//find mD
		// email.Subject = "Lead is awaiting for approval"
		// email.Message = "A lead has been created by marketing executive: " + lead.LeadId
		// sqlStmtFindMD1 := `select emailid FROM dbo.users_master_newpg where role='Managing Director'`
		// rows, err = db.Query(sqlStmtFindMD1)
		// defer rows.Close()
		// for rows.Next() {
		// 	var e string
		// 	err = rows.Scan(&e)
		// 	log.Println("Scanned MD Email is: ", e)
		// 	email.ToEmail = e
		// 	log.Println("sending email")
		// 	smtpSendEmail(generalTemp, email.Subject, email.Message, "", "", "", email.ToEmail)

		// }

		sqlStatementl1 := `UPDATE dbo.CMS_LEADS_MASTER
						  	  SET 
						  	  masterstatus='Pending Approval'
						      WHERE 
					    	  leadid=$1`
		rows, err = db.Query(sqlStatementl1, lead.LeadId)
		log.Println("Updated Status to Pending Approval")

		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		defer rows.Close()

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

		// LEAD REJECTION MODULE
	} else if (lead.Role == "Managing Director") && (lead.Reject) {

		sqlStatementmdr1 := `UPDATE dbo.CMS_LEADS_MASTER
							SET 
							masterstatus='Rejected',
							comments=$1
	 						WHERE 
	  						leadid=$2`
		rows, err = db.Query(sqlStatementmdr1, lead.Comments, lead.LeadId)
		log.Println("Updated Status to Rejected")

		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		defer rows.Close()

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
		// APPROVAL BY MD
	} else if (lead.Role == "Managing Director") && (lead.ConvertLeadToAccount || lead.Approve) {
		log.Println("Entered Lead approval segment")
		//Find latest accidsno
		sqlStatementGACS := `SELECT accountid 
							FROM dbo.accounts_master 
							where accountid is not null
							ORDER BY accountid DESC 
							LIMIT 1`
		rows, err = db.Query(sqlStatementGACS)

		// var po InputAdditionalDetails
		for rows.Next() {
			err = rows.Scan(&lead.LastAccIdsno)
		}
		//get username to set account owner
		log.Println("Finding username of  Lead")

		sqlStatementFU1 := `select u.username from dbo.users_master_newpg u 
								inner join dbo.CMS_LEADS_MASTER ld 
								on ld.createduserid=u.userid 
								where ld.leadid=$1`
		rows, err = db.Query(sqlStatementFU1, lead.LeadId)
		for rows.Next() {
			err = rows.Scan(&lead.UserName)
		}
		//Generating PO NOs----------------
		lead.AccIdsno = lead.LastAccIdsno + 1
		lead.AccountId = strconv.Itoa(lead.AccIdsno)
		autogencode := "Account-" + lead.LeadId
		sqlStatementacn1 := `UPDATE dbo.CMS_LEADS_MASTER
							  SET 
							  masterstatus='Account Created',
					  		  accountid=$1
					  		  WHERE 
					  		  leadid=$2`
		rows, err = db.Query(sqlStatementacn1, lead.AccountId, lead.LeadId)
		log.Println("Account ID assigned")
		sqlStatementina1 := `INSERT INTO dbo.Accounts_master (
			accountid,
			accountname,
			accounttypeid,
			phone,
			email,
			createddate,
			createduserid,
			approxannualrev,
			website,
			productsegmentid,
			recordtypeid,
			shipping_continent,
			shipping_country,
			comments,
			aliases,
			isactive,
			otherinformation)
			SELECT
			accountid,
			accountname,
			accounttypeid,
			phone,
			email,
			createddate,
			createduserid ,
			CAST(approxannualrev as numeric),
			website,
			productsegmentid,
			recordtypeid,
			shipping_continent,
			shipping_country,
			comments,
			aliases,
			isactive,
			otherinformation
			FROM
			dbo.cms_leads_master
			WHERE leadid=$1`
		rows, err = db.Query(sqlStatementina1, lead.LeadId)
		// Get Accountid from Lead Record
		// Set Account status to Prospect in accounts_master
		log.Println("Setting accountowner")
		sqlStatementstat1 := `UPDATE dbo.accounts_master
					 		 SET 
					 		 account_owner=$1,
					 		 masterstatus='Prospect',
							 comments=$2,
							 autogencode=$3
					 		 where accountid=$4`

		_, err = db.Query(sqlStatementstat1, lead.UserName, lead.Comments, autogencode, lead.AccountId)
		if err != nil {
			log.Println(err)
			log.Println("unable to set account owner")
		}
		sqlStatementapp1 := `UPDATE dbo.CMS_LEADS_MASTER
					  	  SET 
					  	  masterstatus='Appoved'
						  comments=$1	
					      WHERE 
					      leadid=$2`

		rows, err = db.Query(sqlStatementapp1, lead.Comments, lead.LeadId)
		fmt.Println("Lead Status is set to Approved")
		//Get contactid from the above insertion
		var latestconid string
		var conid int

		sqlStatementcon2 := `select contactid from dbo.contacts_master order by contactid desc limit 1`
		rows, err = db.Query(sqlStatementcon2)
		if err != nil {
			log.Println(err)
			log.Println("unable to find latest contactid")
		}
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&conid)
		}
		latestconid = strconv.Itoa(conid + 1)
		log.Println("Scanned latest contactid:", latestconid)
		log.Println("Inserting into Contacts Table")
		sqlStatementcon1 := `insert into dbo.contacts_master(
			contactid,
			contactfirstname,
			contactlastname,
			contactemail,
			contactphone,
			contactmobilenumber,
			accountid,
			position,
			salutationid,
			contactcreatedby) 
			select
			$1,
			contactfirstname,
			contactlastname,
			email,
			phone,
			contact_mobile,
			accountid,
			contact_position,
			contact_salutationid,
			createduserid
			from
			dbo.cms_leads_master where leadid=$2`
		rows, err = db.Query(sqlStatementcon1, latestconid, lead.LeadId)
		log.Println("Lead Contact data is inserted into Contacts_Master Successfully")

		//Insert into accounts_billing_address_master
		sqlStatementabm1 := `insert into dbo.accounts_billing_address_master(
			contactid,
			accountid,
			billingid,
			street,
			city,
			stateprovince,
			postalcode,
			country)
			select
			$1,
			ld.accountid,
			lba.billingid,
			lba.street,
			lba.city,
			lba.stateprovince,
			lba.postalcode,
			lba.country
			from
			dbo.cms_leads_billing_address_master lba
			inner join
			dbo.cms_leads_master ld on ld.leadid=lba.leadid
			where ld.leadid=$2`
		rows, err = db.Query(sqlStatementabm1, latestconid, lead.LeadId)
		fmt.Println("Account Contact data is inserted into accounts_billing_address_master")
		//

		//Insert into accounts_shipping_address_master
		sqlStatementasm1 := `insert into dbo.accounts_shipping_address_master(
								contactid,
								accountid,
								shippingid,
								street,
								city,
								stateprovince,
								postalcode,
								country)
								select
								$1,
								ld.accountid,
								lsa.shippingid,
								lsa.street,
								lsa.city,
								lsa.stateprovince,
								lsa.postalcode,
								lsa.country
								from
								dbo.cms_leads_shipping_address_master lsa
								inner join
								dbo.cms_leads_master ld on ld.leadid=lsa.leadid
								where ld.leadid=$2`
		rows, err = db.Query(sqlStatementasm1, latestconid, lead.LeadId)
		fmt.Println("Account Contact data is inserted into dbo.accounts_shipping_address_master")

		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		defer rows.Close()
		//Send email

		// email.Subject = "Lead Approved"
		// email.Message = "Lead is converted into account successfully: " + lead.LeadId
		// email.ToEmail = lead.UserEmail
		// smtpSendEmail(generalTemp, email.Subject, email.Message, "", "", "", email.ToEmail)
		// //Insert Notification
		sqlStatementNotif := `insert into dbo.notifications_master_newpg(userid,feature_category,status) 
								values((select createduserid FROM
											dbo.cms_leads_master
									WHERE leadid=$1),('Account'),('Lead converted into Account'))`
		_, _ = db.Query(sqlStatementNotif, lead.LeadId)

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}

	res1, _ := json.Marshal("Success")
	return events.APIGatewayProxyResponse{200, headers, nil, string(res1), false}, nil
}

func main() {
	lambda.Start(insertLeadIntoAccount)
}
// func smtpSendEmail(temp, subject, message, accname, acccountry, accowner, to_email string) (string, error) {
// 	log.Println("Entered SMTP Email Module")
// 	// Receiver email address.
// 	to := []string{
// 		to_email,
// 	}
// 	// Authentication.
// 	auth := smtp.PlainAuth("", from_email, smtp_pass, smtpHost)

// 	t := template.Must(template.New(temp).Parse(temp))

// 	var body bytes.Buffer

// 	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
// 	body.Write([]byte(fmt.Sprintf("Subject:"+subject+"\n%s\n\n", mimeHeaders)))
// 	//   body.Write([]byte(fmt.Sprintf("Subject: This is a test subject \n%s\n\n", mimeHeaders)))

// 	t.Execute(&body, struct {
// 		EMessage    string
// 		EAccName    string
// 		EAccCountry string
// 		EAccOwner   string
// 	}{

// 		EMessage:    message,
// 		EAccName:    accname,
// 		EAccCountry: acccountry,
// 		EAccOwner:   accowner,
// 	})
// 	log.Println("Sending email to: ", to_email)
// 	// Sending email.
// 	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to, body.Bytes())
// 	log.Println("Email sent")
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	return "Email Sent!", nil
// }

// func addNotif(){
// 	//Insert Notification
// 	sqlStatementNotif := `insert into dbo.notifications_master_newpg(userid,feature_category,status)
// 	values($1,'Lead','Lead Created')`
// 	_, err = db.Query(sqlStatementNotif,lead.CreatedUserid)

// 	sqlStatement2 := `SELECT leadid FROM dbo.cms_leads_master where accountname=$1`

// 	rows, err = db.Query(sqlStatement2, lead.Accountname)

// 	if err != nil {
// 		log.Println(err.Error())
// 		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
// 	}
// }
