package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"text/template"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	from_email = "itsupport@continental.coffee"
	smtp_pass  = "is@98765"
	smtpHost   = "smtp.gmail.com"
	smtpPort   = "587"
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

type InputDetails struct {
	ToEmail string `json:"to_email"`
	ToName  string `json:"name"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func sendITSupportEmail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	var inputDetails InputDetails

	err := json.Unmarshal([]byte(request.Body), &inputDetails)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	response, err := smtpSendEmail(generalTemp, inputDetails.Subject, inputDetails.Message, inputDetails.ToEmail)
	res, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(sendITSupportEmail)
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
