package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)
//connection to database
const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)
//creating vendordetails structre
type TaxMasterDetails struct {
	TaxCreate         bool   `json:"create"`
	TaxUpdate         bool   `json:"update"`
	TaxView           bool   `json:"view"`
	TaxId        string             `json:"tax_id"`
	Type         string `json:"type"`
	FileName     string `json:"file_name"`
	DocKind      string `json:"doc_kind"`
	DocId        string `json:"docid"`
	DocumentName string `json:"document_name"`
	FileContent  string `json:"document_content"`
	CreatedUserid  string `json:"createduserid"`
	ModifiedUserID string `json:"modifieduserid"`
	TaxName         string             `json:"tax_name"`
	TaxPercentage 	string             `json:"tax_percentage"`
	IsActive		bool 			   `json:"isactive"`
	TaxNotes 		string			   `json:"tax_notes"`
	
}	

type AuditLogSupplier struct {
	CreatedDate    string `json:"createddate"`
	CreatedUserid  string `json:"createduserid"`
	ModifiedDate   string `json:"modifieddate"`
	ModifiedUserid string `json:"modifieduserid"`
	Description    string `json:"description"`
}
type FileResponse struct {
	FileName        string `json:"fileName"`
	FileLink        string `json:"fileLink"`
	FileData        string `json:"fileData"`
	FileContentType string `json:"fileContentType"`
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


type NullString struct {
	sql.NullString
}

// MarshalJSON for NullString
func (ns *NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}
var Files_Upload_Loc = os.Getenv("S3_TAX_MASTER_LOC")
var PsqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
var rows *sql.Rows
func cvu_Tax_Master(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-taxuested-With, Content-Type, Accept"}
	log.Println("check1")
	var tax TaxMasterDetails	
	err := json.Unmarshal([]byte(request.Body), &tax)
	db, err := sql.Open("postgres", PsqlInfo)
	if err != nil {
		log.Println("before db ping")
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()
	// check db
	err = db.Ping()
	if err != nil {
		log.Println("after db ping")
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	fmt.Println("Connected!")
	// var rows *sql.Rows
	if tax.TaxCreate {
		var newidsno int
		//Find latest vendor id		
		log.Println("Found exisisting record")
		//Generating vendor NOs
		newidsno=findLatestSerial("taxidsno","dbo.project_tax_master_newpg","taxidsno","taxidsno")
		tax.TaxId= "TAX-" + strconv.Itoa(newidsno)		
		sqlStatementInsTax := `INSERT INTO dbo.project_tax_master_newpg(
			TaxId, TaxIdsno, taxname,percentage, isactive,createdby,createddate)
			VALUES ($1,$2,$3,$4,$5,$6,now())`
		_, errInsTax := db.Query(sqlStatementInsTax,
			tax.TaxId,
			newidsno,
			tax.TaxName,
			tax.TaxPercentage,
			tax.IsActive,
			tax.CreatedUserid)
		log.Println("Insert into Tax master Executed")
		if errInsTax != nil {
			log.Println(errInsTax.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errInsTax.Error(), false}, nil
		}
		//Insert into notes table-Generic
		newnoteids:=findLatestSerial("idsno","dbo.notes_master","idsno","idsno")
		noteid:="NoteID-"+ strconv.Itoa(newnoteids)
		sqlStatementInsTaxNotes := `INSERT INTO dbo.notes_master(
			notes_id, notes_content, notes_sourceid, idsno, createduserid, createddate)
			VALUES ($1, $2, $3, $4, $5,now());`
		_, err := db.Query(sqlStatementInsTaxNotes,
			noteid,
			tax.TaxNotes,
			tax.TaxId,
			newnoteids,
			tax.CreatedUserid)
		log.Println("Insert into Tax master notes")
		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil,"There was a problem while saving notes", false}, nil
		}		
	} else if tax.TaxUpdate && tax.TaxId != "" {
		log.Println("Entered update tax")
		sqlStatementUpTax := `update dbo.project_tax_master_newpg
								set
								taxname=$1,
								percentage=$2,
								isactive=$3,
								modifiedby=$4,
								modifieddate=now()
						 		 where taxid = $5`
		_, err = db.Query(sqlStatementUpTax,
							tax.TaxName,
							tax.TaxPercentage,
							tax.IsActive,
							tax.ModifiedUserID,
							tax.TaxId)
		log.Println("just before error nil check")
		if err != nil {
			log.Println("unable to update tax details", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		if tax.FileContent != "" {
			fileName := "Tax_Master_" + tax.TaxId + ".pdf"
			newattachids:=findLatestSerial("idsno","dbo.attachments_master","idsno","idsno")
			attachmentid:="Tax_DocID-"+ strconv.Itoa(newattachids)
			sqlStatementUTD := `INSERT INTO dbo.attachments_master(
				attachment_id, attachment_filename, attachment_sourceid, idsno, createduserid, createddate)
			VALUES ($1,$2,$3,$4,$5,now());`
			
			_, err = db.Query(sqlStatementUTD, attachmentid, tax.FileName, tax.TaxId,newattachids,tax.CreatedUserid)

			if err != nil {
				log.Println("Attachment upload into S3 failed",err.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, "Attachment upload into S3 failed", false}, nil
			}
			log.Println("Successfully uploaded file in db with ")

			k, err := uploadDocToS3(tax.FileContent, fileName)
			if err != nil {
				log.Println("unable to upload in s3 bucket", err)
			}		
			log.Println("Successfully uploaded file in s3 bucket ", k, fileName)
			// return events.APIGatewayProxyResponse{200, headers, nil, string(fileName), false}, nil
		}
		if tax.TaxNotes !=""{
			//find if tax has notes or not
			sqlFindNote:=`select notes_id from dbo.notes_master where notes_sourceid=$1`
			rows, err = db.Query(sqlFindNote,tax.TaxId)
			var notes_id string
			for rows.Next() {			
				err = rows.Scan(&notes_id)
			}
			if notes_id!=""{
				log.Println("Entered update tax Notes")
				sqlStatementUpTaxNote := `update dbo.notes_master
								set
								notes_content=$1
						 		 where notes_sourceid = $2`
				_, err = db.Query(sqlStatementUpTaxNote,
							tax.TaxNotes,
							tax.TaxId)
				log.Println("Notes update successful")
				if err != nil {
					log.Println("unable to update tax notes details", err.Error())
					return events.APIGatewayProxyResponse{500, headers, nil,"Unable to update tax notes details", false}, nil
				}//end of update

			} else {
				//Insert into notes table-Generic
				newnoteids:=findLatestSerial("idsno","dbo.notes_master","idsno","idsno")
				noteid:="NoteID-"+ strconv.Itoa(newnoteids)
				sqlStatementInsTaxNotes := `INSERT INTO dbo.notes_master(
				notes_id, notes_content, notes_sourceid, idsno, createduserid, createddate)
				VALUES ($1, $2, $3, $4, $5,now());`
				_, err := db.Query(sqlStatementInsTaxNotes,
				noteid,
				tax.TaxNotes,
				tax.TaxId,
				newnoteids,
				tax.CreatedUserid)
				log.Println("Insert into Tax master notes")
				if err != nil {
					log.Println(err.Error())
					return events.APIGatewayProxyResponse{500, headers, nil,"There was a problem while saving notes", false}, nil
				}
			} 	
		}
		
	} else if tax.TaxView && tax.TaxId != "" {
		sqlStatementTaxView := `select tx.taxname,tx.percentage, tx.isactive,note.notes_content
								from dbo.project_tax_master_newpg tx
								left join dbo.notes_master note on note.notes_sourceid=tx.taxid
								where TaxId=$1`
		rowsTaxView, errTaxView := db.Query(sqlStatementTaxView, tax.TaxId)
		log.Println("fetch query executed")
		if errTaxView != nil {
			log.Println("Query failed")
			log.Println(errTaxView.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errTaxView.Error(), false}, nil
		}
		var taxname,percentage,notes_content sql.NullString
		for rowsTaxView.Next() {			
			errTaxView = rowsTaxView.Scan(&taxname,&percentage,&tax.IsActive,&notes_content)
			tax.TaxName = taxname.String
			tax.TaxPercentage=percentage.String
			tax.TaxNotes=notes_content.String
			log.Println("next inside")
		}
		res, _ := json.Marshal(tax)
		log.Println("came out")
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else if tax.Type == "uploadDocument" {

		
	} else if tax.Type == "removeDocument" {
		sqlStatementRTD := `delete from dbo.dbo.attachments_master where attachment_sourceid=$1`
		_, err = db.Query(sqlStatementRTD, tax.TaxId)
		if err != nil {
			log.Println("Error in removing the document attachment",err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil,"Error in removing the document attachment", false}, nil
		}
		log.Println("Successfully removed file in db with ", tax.FileName)
		return events.APIGatewayProxyResponse{200, headers, nil, string("Removed Successfully"), false}, nil
	} else if tax.Type == "downloadDocument" {
		log.Println("starting downloaded ", tax.FileName)
		fileResponse := DownloadFile(tax.FileName)
		log.Println("Successfully downloaded ", tax.FileName)
		response, err := json.Marshal(fileResponse)
		if err != nil {
			log.Println(err.Error())
		}
		return events.APIGatewayProxyResponse{200, headers, nil, string(response), false}, nil
	}
	return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
}
func main() {
	lambda.Start(cvu_Tax_Master)
}
func findLatestSerial(param1, param2, param3, param4 string) (ids int) {
	log.Println("Finding latest serial num")
	db, _ := sql.Open("postgres", PsqlInfo)
	defer db.Close()
	var rows *sql.Rows
	sqlStatement1 := fmt.Sprintf("SELECT %s FROM %s where %s is not null ORDER BY %s DESC LIMIT 1", param1, param2, param3, param4)
	rows, err := db.Query(sqlStatement1)
	for rows.Next() {
		err = rows.Scan(&ids)
	}
	if err != nil {
		log.Println(err)
	}
	return ids + 1
}
func Base64Encoder(s3Client *s3.S3, link string) string {
	tax := &s3.GetObjectInput{
		Bucket: aws.String(Files_Upload_Loc),
		Key:    aws.String(link),
	}
	result, err := s3Client.GetObject(tax)
	if err != nil {
		log.Println(err.Error())
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(result.Body)
	fmt.Println(buf)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
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