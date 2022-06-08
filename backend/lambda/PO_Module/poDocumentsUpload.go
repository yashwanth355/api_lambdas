package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"

	"os"
	"strconv"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

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

type PoDocumentDetails struct {
	DocId               string `json:"docid"`
	DocumentName        string `json:"document_name"`
	FileName            string `json:"file_name"`
	DocKind             string `json:"doc_kind"`
	Required            bool   `json:"required"`
	DispatchId          string `json:"dispatchid"`
	Billofladdingnumber string `json:"billofladdingnumber"`
	Billofladdingdate   string `json:"billofladdingdate"`
	Billofentrynumber   string `json:"billofentrynumber"`
	Billofentrydate     string `json:"billofentrydate"`
	Invoicenumber       string `json:"invoicenumber"`
	Invoicedate         string `json:"invoicedate"`
	Conversationratio   string `json:"conversationratio"`
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

type LastDocDetails struct {
	DocIdno int `json:"docid_no"`
}

type FileResponse struct {
	FileName        string `json:"fileName"`
	FileLink        string `json:"fileLink"`
	FileData        string `json:"fileData"`
	FileContentType string `json:"fileContentType"`
}

type DocumentsUpload struct {
	DocId               string `json:"docid"`
	Billofladdingnumber string `json:"billofladdingnumber"`
	Billofladdingdate   string `json:"billofladdingdate"`
	Billofentrynumber   string `json:"billofentrynumber"`
	Billofentrydate     string `json:"billofentrydate"`
	Invoicenumber       string `json:"invoicenumber"`
	Invoicedate         string `json:"invoicedate"`
	Conversationratio   string `json:"conversationratio"`
}

type Input struct {
	Type             string            `json:"type"`
	PoId             string            `json:"po_id"`
	FileName         string            `json:"file_name"`
	DocKind          string            `json:"doc_kind"`
	DocId            string            `json:"docid"`
	DocumentName     string            `json:"document_name"`
	FileContent      string            `json:"document_content"`
	Required         bool              `json:"required"`
	UpdatedBy        string            `json:"updatedBy"`
	DocumentsSection []DocumentsUpload `json:"documentsection"`
}

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
		<h3>Hi,{{.EHeaderText}}</h3>
			<p>{{.EMessage}}</p>
		<p>Thanks & Regards,</p>
		<p>IT Support</p>
		</body>
		</html>`

var Files_Upload_Loc = os.Getenv("S3_PO_LOC")

func poDocumentsUpload(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	var rows *sql.Rows

	fmt.Println("Connected!")
	var documentDetail PoDocumentDetails
	if input.Type == "getDocumentsOnPo" {
		sqlStatement := `select docid, docname, filename ,dockind, required, dispatchid,billofladdingnumber,billofladdingdate,billofentrynumber,billofentrydate,invoicenumber, invoicedate, conversationratio from dbo.pur_gc_po_master_documents where poid=$1`
		rows, err = db.Query(sqlStatement, input.PoId)

		if err != nil {
			log.Println("Unable to get files uploaded for specific po")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		var docName, fileName, docKind, required, dispatchId, billofladdingnumber, billofladdingdate, billofentrynumber, billofentrydate, invoicenumber, invoicedate, conversationratio sql.NullString
		defer rows.Close()
		var documents []PoDocumentDetails
		for rows.Next() {
			var dt PoDocumentDetails
			err = rows.Scan(&dt.DocId, &docName, &fileName, &docKind, &required, &dispatchId, &billofladdingnumber,
				&billofladdingdate, &billofentrynumber, &billofentrydate, &invoicenumber, &invoicedate, &conversationratio)
			dt.DocumentName = docName.String
			dt.FileName = fileName.String
			dt.DocKind = docKind.String
			dt.DispatchId = dispatchId.String
			dt.Billofladdingnumber = billofladdingnumber.String
			dt.Billofladdingdate = billofladdingdate.String
			dt.Billofentrynumber = billofentrynumber.String
			dt.Billofentrydate = billofentrydate.String
			dt.Invoicenumber = invoicenumber.String
			dt.Invoicedate = invoicedate.String
			dt.Conversationratio = conversationratio.String
			dt.Required, _ = strconv.ParseBool(required.String)

			documents = append(documents, dt)
		}

		res, _ := json.Marshal(documents)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else if input.Type == "uploadDocument" {

		fileName := "Document_GC_" + input.DocId + ".pdf"

		sqlStatement1 := `update dbo.pur_gc_po_master_documents set docname=$1, filename=$2 where docid=$3`
		rows, err = db.Query(sqlStatement1, input.DocumentName, fileName, input.DocId)

		if err != nil {
			log.Println("update to po document master failed")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		log.Println("Successfully uploaded file in db with ", input.DocId, fileName)

		k, err := uploadDocToS3(input.FileContent, fileName)
		if err != nil {
			log.Println("unable to upload in s3 bucket", err)
		}
		if k != "" && err == nil {
			sqlStatement := `select p.pono,v.username,v.emailid, u.username from dbo.pur_gc_po_con_master_newpg p
			left join dbo.users_master_newpg v on p.createdby= v.userid
			left join dbo.users_master_newpg u on u.userid=$1 where p.poid=$2`
			rows2, err2 := db.Query(sqlStatement, input.UpdatedBy, input.PoId)

			if err2 != nil {
				log.Println("Unable to get user details")
			}

			var pono, purchaseOwnerName, purchaseOwnerEmailId, uploadedUserName sql.NullString
			for rows2.Next() {
				err = rows2.Scan(&pono, &purchaseOwnerName, &purchaseOwnerEmailId, &uploadedUserName)
			}
			if purchaseOwnerEmailId.Valid {
				subject := "Documents have been provided for " + pono.String
				body := "The following document " + "https://qa.cclproducts.com/purchase-orders" + " has been uploaded by " + uploadedUserName.String + " for " + pono.String
				_, err = smtpSendEmail(generalTemp, subject, body, purchaseOwnerName.String, purchaseOwnerEmailId.String)
				if err != nil {
					log.Println("Unable to send email to PO owner", purchaseOwnerName.String)
				}
			}
		}
		log.Println("Successfully uploaded file in s3 bucket ", k, fileName)
		return events.APIGatewayProxyResponse{200, headers, nil, string(fileName), false}, nil
	} else if input.Type == "removeDocument" {

		sqlStatement := `update dbo.pur_gc_po_master_documents set docname='', filename='' where docid=$1`
		rows, err = db.Query(sqlStatement, input.DocId)

		log.Println("Successfully removed file in db with ", documentDetail.FileName)
		return events.APIGatewayProxyResponse{200, headers, nil, string("Removed Successfully"), false}, nil
	} else if input.Type == "downloadDocument" {

		log.Println("starting downloaded ", input.FileName)
		fileResponse := DownloadFile(input.FileName)
		log.Println("Successfully downloaded ", input.FileName)
		response, err := json.Marshal(fileResponse)
		if err != nil {
			log.Println(err.Error())
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string(response), false}, nil
	} else if input.Type == "updatedocumentsInfo" {

		log.Println("Going to update documents info", input.DocumentsSection)
		if input.DocumentsSection != nil && len(input.DocumentsSection) > 0 {
			for i, document := range input.DocumentsSection {
				log.Println("document in loop", i, document)

				sqlStatement1 := `Update dbo.pur_gc_po_master_documents set billofladdingnumber=$1,billofladdingdate=$2,billofentrynumber=$3,billofentrydate=$4,
				invoicenumber=$5, invoicedate=$6, conversationratio=$7 where docid=$8`
				rows, err = db.Query(sqlStatement1, NewNullString(document.Billofladdingnumber), NewNullString(document.Billofladdingdate), NewNullString(document.Billofentrynumber), NewNullString(document.Billofentrydate),
					NewNullString(document.Invoicenumber), NewNullString(document.Invoicedate), NewNullString(document.Conversationratio), document.DocId)
				if err != nil {
					log.Println("Unable to update documents info", err.Error())
				}

			}
			return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
		}
	}
	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(poDocumentsUpload)
}

func uploadDocToS3(data string, fileDir string) (string, error) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-south-1"),
	})

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Println(err)
		return "", err
	}

	s3Output, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(Files_Upload_Loc),
		Key:    aws.String(Files_Upload_Loc + "/" + fileDir),
		Body:   bytes.NewReader(dec),
	})
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println(s3Output)
	log.Println("fileLocation: " + s3Output.Location)
	return s3Output.Location, nil
}

func DownloadFile(fileName string) FileResponse {
	// The session the S3 Uploader will use
	svc := s3.New(session.New())

	var fileResponse FileResponse
	fileResponse.FileData = Base64Encoder(svc, Files_Upload_Loc+"/"+fileName)
	fileResponse.FileName = fileName
	fileResponse.FileContentType = "application/pdf"

	return fileResponse
}

func Base64Encoder(s3Client *s3.S3, link string) string {
	input := &s3.GetObjectInput{
		Bucket: aws.String(Files_Upload_Loc),
		Key:    aws.String(link),
	}
	result, err := s3Client.GetObject(input)
	if err != nil {
		log.Println(err.Error())
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(result.Body)
	fmt.Println(buf)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func smtpSendEmail(temp, subject, message, headerText, to_email string) (string, error) {
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
		EMessage    string
		EHeaderText string
	}{
		EMessage:    message,
		EHeaderText: headerText,
	})

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to, body.Bytes())
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return "Email Sent!", nil
}
