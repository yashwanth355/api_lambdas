package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	_ "github.com/lib/pq"

	"bytes"
	"net/smtp"
	"text/template"
)

const (
	ATTACHMENT_TYPE_PDF = "application/pdf"

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

const poTemp = `<!DOCTYPE html>
	    <html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		</head>
		<body>
			<h3>Hello {{.EName}},</h3>
			<p>{{.EMessage}}</p>
			<p>Regards,</p>
			<p>{{.EDept}}</p>
		</body>
	</html>`
const vendorTemp = `<!DOCTYPE html>
	<html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		</head>
		<body>
			<h3>Hello {{.EName}},</h3>
			<p>You are requested to provide the green coffee specification details for Purchase Order : {{.PONO}}</p>
			<p>Please click the below link :</p>
			<p>{{.VendorURL}}</p>
			<p>Use the following credentials </p>
			<p>UserId: {{.EVendoremailid}} </p>
			<p>OTP: Vendor@1234 </p>
			<p>Regards,</p>
			<p>CCL Purchase Department</p>
		</body>
	</html>`

// {{.VEmail}}

// Email is input request data which is used for sending email using aws ses service
type Email struct {
	ToEmail   string `json:"to_email"`
	ToName    string `json:"name"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	VendorURL string `json:"vendor_url"`
}

type Input struct {
	Type          string `json:"type"`
	CreatedUserID string `json:"createduserid"`
	PoId          string `json:"po_id"`
	PoNO          string `json:"po_no"`
	VendorEmail   string `json:"vendor_email"`
	SendEmail     bool   `json:"notify_email"`
	UserEmail     string `json:"emailid"`

	VendorDocFileName string `json:"document_name"`
	VendorDocContent  string `json:"document"`
}

func updateGcPoStatus(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input Input
	var email Email
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
	// send email to email id provided in request when it is send email
	vendorEmailIdFromRequest := input.VendorEmail
	if input.PoId != "" {
		log.Println("Selected POId is : ", input.PoId)
		sqlStatementGE := `select po.pono,initcap(ven.vendorname),ven.email from
							dbo.pur_gc_po_con_master_newpg po
							right join dbo.pur_vendor_master_newpg ven
							on ven.vendorid=po.vendorid
							where po.poid=$1`

		rowsGE, errGE := db.Query(sqlStatementGE, input.PoId)
		if errGE != nil {
			log.Println(errGE)
			return events.APIGatewayProxyResponse{500, headers, nil, errGE.Error(), false}, nil
		}
		defer rowsGE.Close()
		for rowsGE.Next() {
			errGE = rowsGE.Scan(&input.PoNO, &email.ToName, &input.VendorEmail)
		}
		log.Println("Scanned Vendor email is :", input.VendorEmail)
		//Email Module
		//Email is triggered when PO is Approved

		if input.Type == "None" {
			sqlStatement := `update dbo.pur_gc_po_con_master_newpg set status ='2' where poid=$1`
			_, err := db.Query(sqlStatement, input.PoId)

			if err != nil {
				log.Println("Unable to change status to pending state")
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}

			return events.APIGatewayProxyResponse{200, headers, nil, string("Successfully changed"), false}, nil
		} else if input.Type == "changeToInprogessStatus" {
			email.ToEmail = input.VendorEmail
			// ToAddressEmail := []string{input.UserEmail+","+input.VendorEmail}

			sub := "Link to submit Green Coffee Specification for CCL Green Coffee Purchase Order: " + input.PoNO
			email.Message = "PO Status has been changed"
			email.VendorURL = os.Getenv("GC_PO_Vendor_URL")
			if input.VendorEmail != "" {
				smtpSendEmail(email.VendorURL, vendorTemp, sub, email.ToName, email.Message, input.PoNO, "PO Department", email.ToEmail, input.VendorEmail)
			}

			sqlStatement := `update dbo.pur_gc_po_con_master_newpg set status ='3' where poid=$1`
			_, err := db.Query(sqlStatement, input.PoId)

			if err != nil {
				log.Println("Unable to change status to in progess state")
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}
			log.Println("Entered Email Trigger Module")
			if input.VendorEmail != "" {

			}
			return events.APIGatewayProxyResponse{200, headers, nil, string("Successfully changed"), false}, nil
		} else if input.Type == "changeToPendingStatus" {
			// Sending email.
			email.ToEmail = input.UserEmail
			email.Subject = "PO has been sent for approval: " + input.PoNO
			email.Message = "PO Status has been changed"
			email.VendorURL = os.Getenv("GC_PO_Vendor_URL")
			smtpSendEmail(email.VendorURL, poTemp, email.Subject, email.ToName, email.Message, input.PoNO, "PO Department", email.ToEmail, "")
			log.Println("Entered Status Change Module")
			sqlStatement := `update dbo.pur_gc_po_con_master_newpg set status ='2' where poid=$1`
			_, err := db.Query(sqlStatement, input.PoId)

			if err != nil {
				log.Println("Unable to change status to pending state")
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}

			return events.APIGatewayProxyResponse{200, headers, nil, string("Status Successfully changed and Email Sent to Vendor"), false}, nil

		} else if input.Type == "close" {

			log.Println("Entered Close Status Change Module")
			sqlStatement := `update dbo.pur_gc_po_con_master_newpg set status ='6' where poid=$1`
			_, err := db.Query(sqlStatement, input.PoId)

			if err != nil {
				log.Println("Unable to change status to Closed state")
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}

			return events.APIGatewayProxyResponse{200, headers, nil, string("PO Status is set to Closed"), false}, nil

		} else if input.SendEmail {

			log.Println("User requested to send email to vendor with PDF doc to ask spec for PO: ", input.PoNO)
			var errMsg string
			var errCode int

			if input.VendorEmail == "" {
				log.Println("Email id not found in request ")
				errMsg = "Email Id is missing for the GC Supplier/Vendor."
				errCode = 400
			} else {

				updateStatusQuery := `update dbo.pur_gc_po_con_master_newpg set status ='3' where poid=$1`
				_, err := db.Query(updateStatusQuery, input.PoId)
				log.Println("pur_gc_po_con_master_newpg updated with status = 3 for PO: ", input.PoNO)

				if err == nil {
					dataFeedToTemplate := make(map[string]string)
					dataFeedToTemplate["PONumber"] = input.PoNO
					dataFeedToTemplate["VendorName"] = email.ToName
					dataFeedToTemplate["VendorEmail"] = input.VendorEmail
					dataFeedToTemplate["VendorURL"] = os.Getenv("GC_PO_Vendor_URL")
					subject := "Link to submit Green Coffee Specification for CCL Green Coffee Purchase Order: " + input.PoNO
					err = sendAwsSesTemplatedEmailWithAttachment(dataFeedToTemplate,
						[]string{vendorEmailIdFromRequest}, from_email, "Purchase Team",
						subject, input.VendorDocFileName, input.VendorDocContent,
						ATTACHMENT_TYPE_PDF, nil)
				}
			}
			if err != nil {
				errCode = 500
				errMsg = "An error occured. Please contact support."
				log.Println("Error occuured in processing User's Request to Email Vendor with PDF, to ask spec for PO. Error: ", err.Error())
			}
			if errMsg != "" {
				return events.APIGatewayProxyResponse{errCode, headers, nil, string(errMsg), false}, nil
			}
			// ToAddressEmail := []string{input.UserEmail+","+input.VendorEmail}
		}
	} else {
		return events.APIGatewayProxyResponse{500, headers, nil, string("PO id is missing for the GC Supplier/Vendor."), false}, nil
	}
	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(updateGcPoStatus)
}

func smtpSendEmail(vendorurl, temp, subject, name, message, pono, dept, to_email, vendoremailid string) (string, error) {
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
		EName          string
		EMessage       string
		EDept          string
		PONO           string
		EVendoremailid string
		VendorURL      string
	}{
		EName:          name,
		EMessage:       message,
		EDept:          dept,
		PONO:           pono,
		EVendoremailid: vendoremailid,
		VendorURL:      vendorurl,
	})

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to, body.Bytes())
	if err != nil {
		fmt.Println(err)

	}
	return "Email Sent!", nil
}

/*
*
 */
func sendAwsSesTemplatedEmailWithAttachment(templateDynamicData map[string]string,
	emailToIds []string, fromEmail string, fromName string,
	subject string, attachFileName string, base64AttachmentContent string, attachmentContentType string,
	awsSession *session.Session) error {

	log.Println("Entered sendAwsSesTemplatedEmailWithAttachment..")
	var processingErr error
	if awsSession == nil {
		awsSession, processingErr = session.NewSession(&aws.Config{
			Region: aws.String("ap-south-1"),
		})
	}
	awsSesEmailTemplate := "AskSpecificationToVendorForPO"
	if processingErr == nil {
		input := &ses.GetTemplateInput{
			TemplateName: &awsSesEmailTemplate,
		}
		getTemplateOutput, processingErr := ses.New(awsSession).GetTemplate(input)
		if processingErr == nil {
			templateContent := *getTemplateOutput.Template.HtmlPart
			if templateDynamicData != nil || len(templateDynamicData) > 0 {
				for key, val := range templateDynamicData {
					findWhat := "{{" + key + "}}"
					templateContent = strings.Replace(templateContent, findWhat, val, -1)
				}
			}
			var rawMsg string
			rawMsg, processingErr = makeRawMessage(emailToIds, subject,
				templateContent, fromEmail, fromName,
				attachFileName, base64AttachmentContent, attachmentContentType)
			if processingErr == nil {
				//fmt.Println("Raw essage: \n\n", rawMsg)
				rawMsgInput := &ses.SendRawEmailInput{
					RawMessage: &ses.RawMessage{
						Data: []byte(rawMsg),
					},
				}
				_, processingErr = ses.New(awsSession).SendRawEmail(rawMsgInput)
			}
		}
	}
	return processingErr
}

/*
*
 */
func makeRawMessage(emailToIds []string, subject string, message string,
	emailFromId string, emailFromName string,
	attachFileName string, base64AttachmentContent string, attachmentContentType string) (string, error) {

	var buildErr error = nil
	var builder strings.Builder
	if emailFromName == "" {
		emailFromName = emailFromId
	}
	builder.WriteString("From: '" + emailFromName + "' <" + emailFromId + ">\n")
	builder.WriteString("Subject: " + subject + "\n")
	builder.WriteString("To: " + strings.Join(emailToIds[:], ",") + "\n")
	builder.WriteString("MIME-Version: 1.0\n")

	rootBoundaryId := fmt.Sprint(crc32.ChecksumIEEE([]byte("BOUNDARY_FOR_MSG_WITH_ATTACH")))
	builder.WriteString("Content-Type: multipart/mixed; boundary=\"" + rootBoundaryId + "\"\n\n")
	if rootBoundaryId != "" {
		builder.WriteString("--" + rootBoundaryId + "\n")
	}
	builder.WriteString("Content-Type: text/html; charset=UTF-8\n")
	builder.WriteString("Content-Transfer-Encoding: quoted-printable\n\n")
	builder.WriteString(message + "\n")
	// 1 attachment
	builder.WriteString("--" + rootBoundaryId + "\n")
	builder.WriteString("Content-Type: application/pdf; name=\"" + attachFileName + "\"\n")
	builder.WriteString("Content-Disposition: attachment;filename=\"" + attachFileName + "\"\n")
	builder.WriteString("Content-Transfer-Encoding: base64\n")
	builder.WriteString(base64AttachmentContent)
	builder.WriteString("\n")
	builder.WriteString("\n--" + rootBoundaryId + "--")
	return builder.String(), buildErr
}
