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

	//SES
	"bytes"
	"net/smtp"
	"text/template"
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

type InputDetails struct {
	LeadID         string `json:"leadid"`
	UserID         string `json:"userid"`
	LoggedInUserId string `json:"loggedinuserid"`
}

func reassignLead(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	var id InputDetails
	var email Email
	err := json.Unmarshal([]byte(request.Body), &id)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()

	var role string
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	var rows1 *sql.Rows
	sqlStatementAUser1 := `SELECT role FROM dbo.users_master_newpg
							where userid=$1`
	rows1, err = db.Query(sqlStatementAUser1, id.LoggedInUserId)
	for rows1.Next() {
		err = rows1.Scan(&role)
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	var rows *sql.Rows
	fmt.Println("Connected!")
	if id.UserID != "" {
		sqlStatementAUser1 := `SELECT emailid FROM dbo.users_master_newpg
							where userid=$1`
		rows, err = db.Query(sqlStatementAUser1, id.UserID)
		for rows.Next() {
			err = rows.Scan(&email.ToEmail)
		}
		if role == "Managing Director" {
			sqlStatementre1 := `update dbo.cms_leads_master
						set
						createduserid=$1					 
						where leadid=$2`

			_, err = db.Query(sqlStatementre1, id.UserID, id.LeadID)
			defer rows.Close()
			//Send email
			if email.ToEmail != "" {
				email.Subject = "Lead reassigned"
				email.Message = "Lead has been reassigned"
				// email.ToEmail= lead.UserEmail
				smtpSendEmail(generalTemp, email.Subject, email.Message, email.ToEmail)
			}
			res, _ := json.Marshal("Reassign Successful")
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
		} else if role != "Managing Director" {
			res, _ := json.Marshal("Permission denied, you dont have the permission.")
			return events.APIGatewayProxyResponse{402, headers, nil, string(res), false}, nil
		}
	}

	res, _ := json.Marshal("Reassign failed")
	return events.APIGatewayProxyResponse{500, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(reassignLead)
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
	//   body.Write([]byte(fmt.Sprintf("Subject: This is a test subject \n%s\n\n", mimeHeaders)))

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
