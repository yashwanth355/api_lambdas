package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ses"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"

	DEBIT_NOTE_DOC_AUDIENCE_VEDNOR   = "VENDOR"
	DEBIT_NOTE_DOC_AUDIENCE_ACCOUNTS = "ACCOUNTS"

	ACCOUNT_DEPT_ROLE_TO_EMAIL_DOC = "Accounts Manager"

	EMAIL_DN_DOC_TO_VENDOR_FROM      = "itsupport@continental.coffee" //"purchase@continental.coffee"
	EMAIL_DN_DOC_TO_VENDOR_FROM_NAME = "Purchase"

	EMAIL_DN_DOC_TO_ACCOUNTS_FROM      = "itsupport@continental.coffee" //"purchase@continental.coffee"
	EMAIL_DN_DOC_TO_ACCOUNTS_FROM_NAME = "Purchase"

	DN_DOC_EMAIL_ACCOUNTS_DEFAULT_TO = "itsupport@continental.coffee"
)

type Input struct {
	PoId                 string `json:"po_id"`
	FileName             string `json:"file_name"`
	DocKind              string `json:"doc_kind"`
	DocId                string `json:"docid"`
	AccountsDocumentName string `json:"document_name_accounts"`
	VendorDocumentName   string `json:"document_name_vendor"`

	AccountsFileContent string `json:"document_content_accounts"`
	VendorFileContent   string `json:"document_content_vendor"`

	Required        bool   `json:"required"`
	UpdatedBy       string `json:"updatedBy"`
	Debitnoteid     string `json:"debit_noteid"`
	DebitnoteNumber string `json:"debit_note_num"`
	Mrinid          string `json:"mrin_id"`
	MrinNumber      string `json:"mrin_num"`
	PoNumber        string `json:"po_num"`
	DispatchNumber  string `json:"dispatch_num"`
}

var Files_Upload_Loc = os.Getenv("S3_DebitNote_LOC")

/*
*
 */
func main() {
	lambda.Start(releaseDebitNote)
}

/*
*
 */
func releaseDebitNote(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	var processingStatusMsg string
	var opResultStatus int = 200
	var processingErr error = nil
	var db *sql.DB
	db, processingErr = getDbConnection()
	defer db.Close()
	var input Input
	var docExistsForMrinAndDN bool
	if processingErr == nil {
		processingErr = json.Unmarshal([]byte(request.Body), &input)
		docExistsForMrinAndDN, processingErr = docExists(input, db)
	}
	if processingErr == nil && docExistsForMrinAndDN {
		opResultStatus = 409
		log.Println("Doc Exists For Mrin And DN Id: ", docExistsForMrinAndDN)

	} else if processingErr == nil {
		var docIdNumber int
		docIdNumber, processingErr = newDocIdNumber(db)
		if processingErr == nil {
			processingErr = addNewDebitNotDocumentRecord(input, docIdNumber, db)
			if processingErr == nil {
				log.Println("Added pur_gc_debitnote_master_documents_newpg RECORD with DNDocId Number: ", docIdNumber)
				if input.AccountsFileContent != "" && input.VendorFileContent != "" {
					processingErr = storeAndEmailDocs(input, db)
					if processingErr == nil {
						processingErr = markDebitNoteAsReleased(input, db)
					}
				} else {
					opResultStatus = 401
				}
			}
		}
	}
	if processingErr != nil {
		opResultStatus = 500
		log.Println("An error ocuured: ", processingErr.Error())
	}
	switch opResultStatus {
	case 401:
		processingStatusMsg = "No Document File Content found in request."
	case 409:
		processingStatusMsg = "Debit Note released already."
	case 500:
		processingStatusMsg = "An error ocuured. Please contact support."
	case 200:
		processingStatusMsg = "Debit Note Release request executed successfully."
	default:
		processingStatusMsg = "Release Debit Note request processing completed."
	}
	return events.APIGatewayProxyResponse{opResultStatus, headers, nil, string(processingStatusMsg), false}, nil
}

/*
*
 */
func storeAndEmailDocs(request Input, db *sql.DB) error {

	log.Println("storeAndEmailDocs fired..")
	var processingErr error = nil
	var awsSession *session.Session
	awsSession, processingErr = session.NewSession(&aws.Config{
		Region: aws.String("ap-south-1"),
	})
	var docStoredLocation string
	docStoredLocation, processingErr = storeDebitNoteDocument(DEBIT_NOTE_DOC_AUDIENCE_ACCOUNTS,
		request, awsSession)

	if processingErr == nil {
		log.Println("Uploaded DN Doc for Accounts: ", docStoredLocation, request.AccountsDocumentName)
		processingErr = addContextData(&request, db)
	}
	if processingErr == nil {
		processingErr = emailAccountsTeam(request, db, awsSession)
		if processingErr == nil {
			log.Println("Accounts Email Send Request - Success.")
		} else {
			log.Println("Accounts Email Send Request - Failed. Error: ", processingErr)
		}
	}
	if processingErr == nil {
		docStoredLocation, processingErr = storeDebitNoteDocument(DEBIT_NOTE_DOC_AUDIENCE_VEDNOR,
			request, awsSession)
		if processingErr == nil {
			log.Println("Uploaded DN Doc for Vednor: ", docStoredLocation, request.VendorDocumentName)
			processingErr = emailVendor(request, db, awsSession)
			if processingErr == nil {
				log.Println("Vendor Email Send Request - Success.")
			} else {
				log.Println("Vendor Email Send Request - Failed. Error: ", processingErr)
			}
		}
	}
	return processingErr
}

/*
*
 */
func storeDebitNoteDocument(documentAudience string, request Input, awsSession *session.Session) (string, error) {

	var processingErr error = nil
	var s3Location string
	if documentAudience == DEBIT_NOTE_DOC_AUDIENCE_ACCOUNTS {

		s3Location, processingErr = uploadDocToS3(request.AccountsFileContent,
			request.AccountsDocumentName, awsSession)

	} else if documentAudience == DEBIT_NOTE_DOC_AUDIENCE_VEDNOR {

		s3Location, processingErr = uploadDocToS3(request.VendorFileContent,
			request.VendorDocumentName, awsSession)
	}
	return s3Location, processingErr
}

/*
*
*	check if doc exists for the mrin & debitnote
* 	already in the system or not
*
*
 */
func docExists(request Input, db *sql.DB) (bool, error) {

	var numberOfrecords int
	query := `SELECT count(debitdocid) FROM dbo.pur_gc_debitnote_master_documents_newpg
							where debitnoteid=$1 and mrinid=$2`
	rows, err := db.Query(query, strings.TrimSpace(request.Debitnoteid), strings.TrimSpace(request.Mrinid))
	if err == nil {
		for rows.Next() {
			err = rows.Scan(&numberOfrecords)
		}
		if numberOfrecords == 0 && err == nil {
			return false, nil
		}
	}
	return true, err
}

/*
*
 */
func newDocIdNumber(db *sql.DB) (int, error) {

	var lastDocIdNumber int
	query := `SELECT docidsno FROM dbo.pur_gc_debitnote_master_documents_newpg 
		where docidsno is not null ORDER BY docidsno DESC LIMIT 1`
	rows, err := db.Query(query)
	if err == nil {
		for rows.Next() {
			err = rows.Scan(&lastDocIdNumber)
		}
		if err == nil {
			return lastDocIdNumber + 1, nil
		}
	}
	return lastDocIdNumber, err
}

/*
*
 */
func addNewDebitNotDocumentRecord(request Input, docIdNumber int, db *sql.DB) error {

	insertQuery := `INSERT INTO dbo.pur_gc_debitnote_master_documents_newpg (
		debitnoteid, mrinid, debitdocid, docidsno, accounts_docname, vendor_docname ) 
		values ($1, $2, $3, $4, $5, $6)`

	debitNoteDocumentId := "Debit_Note_Release_Doc-" + strconv.Itoa(docIdNumber)

	_, err := db.Query(insertQuery, request.Debitnoteid, request.Mrinid,
		debitNoteDocumentId, docIdNumber, request.AccountsDocumentName,
		request.VendorDocumentName)

	return err
}

/*
*
 */
func addContextData(request *Input, db *sql.DB) error {

	query := `select DN.debitnoteno, MRIN.mrinno, MRIN.pono, DISP.detid from dbo.inv_gc_debitnote_master_newpg DN, 
	dbo.inv_gc_po_mrin_master_newpg MRIN, dbo.pur_gc_po_dispatch_master_newpg DISP 
	where MRIN.mrinid = DN.mrinid and DISP.pono = MRIN.pono and 
	DN.debitnoteid = $1 and MRIN.mrinid = $2`

	rows, err := db.Query(query, request.Debitnoteid, request.Mrinid)
	if err == nil {
		for rows.Next() {
			err = rows.Scan(&request.DebitnoteNumber, &request.MrinNumber, &request.PoNumber, &request.DispatchNumber)
		}
	}
	return err
}

/*
*
 */
func markDebitNoteAsReleased(request Input, db *sql.DB) error {

	updateQuery := `update dbo.inv_gc_debitnote_master_newpg
		set is_released = $1 where debitnoteid = $2`

	_, err := db.Query(updateQuery, true, request.Debitnoteid)

	return err
}

/*
*
 */
func emailVendor(request Input, db *sql.DB, awsSession *session.Session) error {

	vendorsEmailIds, err := getVendorEmail(request, db)
	subject := "Debit Note# " + request.DebitnoteNumber

	log.Println("Vendor Email Ids :", vendorsEmailIds)

	var htmlMessage strings.Builder
	htmlMessage.WriteString("Dear Sir / Madam,")
	htmlMessage.WriteString("<br><br>Please find attached Debit Note ")
	htmlMessage.WriteString(request.DebitnoteNumber)
	htmlMessage.WriteString(" with respect to PO - ")
	htmlMessage.WriteString(request.PoNumber)
	htmlMessage.WriteString(", MRIN - ")
	htmlMessage.WriteString(request.MrinNumber)
	htmlMessage.WriteString(", Dispatch - ")
	htmlMessage.WriteString(request.DispatchNumber)
	htmlMessage.WriteString(" for your information.")
	htmlMessage.WriteString("<br><br>You are requested to acknowledge the same and arrange Credit Note ASAP and oblige, as it is mandatory requirement according to GST guidelines.")
	htmlMessage.WriteString("<br><br>Non receipt of credit note could delay next payment.")
	htmlMessage.WriteString("<br><br>Your early response will be appreciated.")
	htmlMessage.WriteString("<br><br><u><i>This is a system generated mail. Please do not reply to this email ID.</i></u>")
	htmlMessage.WriteString("<br><br>Thanks & Regards,")
	htmlMessage.WriteString("<br>Purchase Team.")

	err = emailWithAttachment(vendorsEmailIds, subject,
		htmlMessage.String(), EMAIL_DN_DOC_TO_VENDOR_FROM,
		EMAIL_DN_DOC_TO_VENDOR_FROM_NAME, request.VendorDocumentName,
		request.VendorFileContent, awsSession)

	return err
}

/*
*
 */
func emailAccountsTeam(request Input, db *sql.DB, awsSession *session.Session) error {

	accountTeamEmails, err := getAccountsTeamEmails(request, db)

	log.Println("Account Team Emails Ids :", accountTeamEmails)
	subject := "Debit Note# " + request.DebitnoteNumber

	var htmlMessage strings.Builder
	htmlMessage.WriteString("Dear Sir / Madam,")
	htmlMessage.WriteString("<br><br>Please find attached Debit Note ")
	htmlMessage.WriteString(request.DebitnoteNumber)
	htmlMessage.WriteString(" with respect to PO - ")
	htmlMessage.WriteString(request.PoNumber)
	htmlMessage.WriteString(", MRIN - ")
	htmlMessage.WriteString(request.MrinNumber)
	htmlMessage.WriteString(", Dispatch - ")
	htmlMessage.WriteString(request.DispatchNumber)
	htmlMessage.WriteString(" for your information.")
	htmlMessage.WriteString("<br><br><u><i>This is a system generated mail. Please do not reply to this email ID.</i></u>")
	htmlMessage.WriteString("<br><br>Thanks & Regards,")
	htmlMessage.WriteString("<br>Purchase Team.")

	err = emailWithAttachment(accountTeamEmails, subject,
		htmlMessage.String(), EMAIL_DN_DOC_TO_ACCOUNTS_FROM,
		EMAIL_DN_DOC_TO_ACCOUNTS_FROM_NAME, request.AccountsDocumentName,
		request.AccountsFileContent, awsSession)

	return err
}

/*
*
 */
func emailWithAttachment(emailToId []string, subject string,
	message string, emailFromId string, emailFromName string, attachFileName string,
	attachmentContent string, awsSession *session.Session) error {

	var processingErr error = nil
	var rawMsg string
	rawMsg, processingErr = buildRawMessage(emailToId, subject,
		message, emailFromId, emailFromName, attachFileName, attachmentContent)
	if processingErr == nil {
		//log.Println("Raw essage: \n\n", rawMsg)
		rawMsgInput := &ses.SendRawEmailInput{
			RawMessage: &ses.RawMessage{
				Data: []byte(rawMsg),
			},
		}
		_, processingErr = ses.New(awsSession).SendRawEmail(rawMsgInput)
	}
	return processingErr
}

/*
*
 */
func buildRawMessage(emailToIds []string, subject string,
	message string, emailFromId string, emailFromName string,
	attachFileName string, attachmentContent string) (string, error) {

	var buildErr error = nil
	var builder strings.Builder
	var msgPart string
	builder.WriteString(buildTopHeaders(emailToIds, subject, message, emailFromId, emailFromName))
	var rootBoundaryId string = ""
	// for attachment - assumes email is with attachment
	rootBoundaryId, msgPart = buildMixedHeader()
	builder.WriteString(msgPart)
	builder.WriteString(buildHtmlBodyPart(message, rootBoundaryId))
	msgPart, buildErr = buildPartForOneAttachment(attachFileName, attachmentContent, "--"+rootBoundaryId)
	if buildErr == nil {
		builder.WriteString(msgPart)
		builder.WriteString("\n--" + rootBoundaryId + "--")
	}
	if buildErr != nil {
		return "", buildErr
	}
	return builder.String(), nil
}

/*
*
 */
func buildPartForOneAttachment(fileName string,
	attachmentContent string,
	boundary string) (string, error) {

	var builder strings.Builder
	builder.WriteString(boundary + "\n")
	builder.WriteString("Content-Type: application/pdf; name=\"" + fileName + "\"\n")
	builder.WriteString("Content-Disposition: attachment;filename=\"" + fileName + "\"\n")
	builder.WriteString("Content-Transfer-Encoding: base64\n")
	builder.WriteString(attachmentContent)
	builder.WriteString("\n")
	return builder.String(), nil
}

/*
*
 */
func buildHtmlBodyPart(htmlBody string, boundary string) string {

	var builder strings.Builder
	if boundary != "" {
		builder.WriteString("--" + boundary + "\n")
	}
	builder.WriteString("Content-Type: text/html; charset=UTF-8\n")
	builder.WriteString("Content-Transfer-Encoding: quoted-printable\n\n")
	builder.WriteString(htmlBody + "\n")
	return builder.String()
}

/*
*
 */
func buildTopHeaders(emailToIds []string, subject string,
	message string, emailFromId string, emailFromName string) string {

	var builder strings.Builder
	if emailFromName == "" {
		emailFromName = emailFromId
	}
	builder.WriteString("From: '" + emailFromName + "' <" + emailFromId + ">\n")
	builder.WriteString("Subject: " + subject + "\n")
	builder.WriteString("To: " + strings.Join(emailToIds[:], ",") + "\n")
	builder.WriteString("MIME-Version: 1.0\n")
	return builder.String()
}

/*
*
 */
func buildMixedHeader() (string, string) {

	var builder strings.Builder
	rootBoundaryId := generateBoundaryId("MESSAGE-WITH-ATTACHMENTs")
	builder.WriteString("Content-Type: multipart/mixed; boundary=\"" + rootBoundaryId + "\"\n\n")
	return rootBoundaryId, builder.String()
}

/*
*
 */
func generateBoundaryId(inputHint string) string {
	return fmt.Sprint(crc32.ChecksumIEEE([]byte(inputHint)))
}

/*
*
 */
func getDbConnection() (*sql.DB, error) {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err == nil {
		err = db.Ping()
	}
	return db, err
}

/*
*
 */
func uploadDocToS3(fileContent string, fileDir string, sess *session.Session) (string, error) {

	log.Println("Uploading to S3: ", fileDir)
	var processingErr error = nil
	var decodedFileContent []byte
	decodedFileContent, processingErr = base64.StdEncoding.DecodeString(fileContent)
	if processingErr == nil {
		uploader := s3manager.NewUploader(sess)
		var s3Output *s3manager.UploadOutput
		s3Output, processingErr = uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(Files_Upload_Loc),
			Key:    aws.String(Files_Upload_Loc + "/" + fileDir),
			Body:   bytes.NewReader(decodedFileContent),
		})
		if processingErr == nil {
			log.Println("S3 Output: ", s3Output)
			return s3Output.Location, nil
		}
	}
	return "", processingErr
}

/*
*
 */
func getVendorEmail(request Input, db *sql.DB) ([]string, error) {

	var vednorEmailId string
	query := `select email from dbo.pur_vendor_master_newpg vendor, 
		dbo.inv_gc_debitnote_master_newpg debitnote
		where vendor.vendorid = debitnote.vendorid 
		and debitnote.debitnoteid = $1`

	rows, err := db.Query(query, request.Debitnoteid)
	if err == nil {
		for rows.Next() {
			err = rows.Scan(&vednorEmailId)
		}
		if err == nil {
			return []string{vednorEmailId}, nil
		}
	}
	return nil, err
}

/*
*
 */
func getAccountsTeamEmails(request Input, db *sql.DB) ([]string, error) {

	var accountTeamEmailIds []string
	query := `select emailid from dbo.users_master_newpg where role = $1 and active = true`
	rows, err := db.Query(query, ACCOUNT_DEPT_ROLE_TO_EMAIL_DOC)
	defer rows.Close()
	if err == nil {
		for rows.Next() {
			var email string
			if err := rows.Scan(&email); err != nil {
				return accountTeamEmailIds, err
			}
			accountTeamEmailIds = append(accountTeamEmailIds, email)
		}
	}
	if err != nil {
		if len(accountTeamEmailIds) == 0 {
			accountTeamEmailIds = []string{DN_DOC_EMAIL_ACCOUNTS_DEFAULT_TO}
		}
		return accountTeamEmailIds, err
	}
	return accountTeamEmailIds, nil
}
