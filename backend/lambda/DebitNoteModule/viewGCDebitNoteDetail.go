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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	// "github.com/aws/aws-sdk-go/service/s3/s3manager"
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

type DebitnoteDetails struct {
	Debitnoteid     string       `json:"debit_noteid"`
	Debitnoteno     string       `json:"debit_noteno"`
	Debitnotedate   string       `json:"debit_notedate"`
	VendorId        string       `json:"vendor_id"`
	VendorName      string       `json:"vendor_name"`
	Remarks         string       `json:"remarks"`
	InvoiceNo       string       `json:"invoice_no"`
	InvoiceQuantity string       `json:"invoice_qty"`
	Status          string       `json:"status"`
	EntityId        string       `json:"entity_id"`
	Mrinid          string       `json:"mrin_id"`
	MrinNo          string       `json:"mrin_no"`
	Itemid          string       `json:"item_id"`
	ItemName        string       `json:"gcitem_name"`
	ItemDescription string       `json:"itemdesc"`
	VehicleNo       string       `json:"vehicleno"`
	DebitAmount     string       `json:"debit_amount"`
	HscCode         string       `json:"hsc_code"`
	Others          string       `json:"others"`
	MrinDate        string       `json:"mrin_date"`
	PoNo            string       `json:"po_no"`
	InvoiceDate     string       `json:"invoice_date"`
	PoDate          string       `json:"po_date"`
	EntityName      string       `json:"entity_name"`
	VendorAddress   string       `json:"vendor_address"`
	EntityAddress   string       `json:"entity_address"`
	AuditLogDetails []AuditLogGC `json:"audit_log_gc"`
	DebitNoteDocs []ReleasedDebitNote `json:"released_debitnotes"`

	//Special composition Info Section---------------------------
	Husk     string `json:"husk"`
	Quality  string `json:"quality"`
	Netrecd  string `json:"netrecd"`
	Moisture string `json:"moisture"`
	Stones   string `json:"stones"`
}

type ReleasedDebitNote struct {
	FileName     string `json:"file_name"`
	DocKind      string `json:"doc_kind"`
	DocId        string `json:"docid"`
	AccountsDocumentName string `json:"document_name_accounts"`
	VendorDocumentName string `json:"document_name_vendor"`
	
	AccountsFileContent  string `json:"document_content_accounts"`
	VendorFileContent  string `json:"document_content_vendor"`
	
}
type AuditLogGC struct {
	CreatedDate      string `json:"created_date"`
	CreatedUserName  string `json:"created_username"`
	ModifiedDate     string `json:"modified_date"`
	ModifiedUserName string `json:"modified_username"`
	Description      string `json:"description"`
}
type FileResponse struct {
	FileName        string `json:"fileName"`
	FileLink        string `json:"fileLink"`
	FileData        string `json:"fileData"`
	FileContentType string `json:"fileContentType"`
}

type Input struct {
	Debitnoteid string `json:"debit_noteid"`
	Type 		string `json:"type"`
	FileName     string `json:"file_name"`
}
var Files_Upload_Loc = os.Getenv("S3_DebitNote_LOC")
func viewGCDebitNoteDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	fmt.Println("Connected!")
	if input.Type=="downloadDebitNote" && input.FileName !="" {
		log.Println("starting download ")
		
		fileResponse := DownloadFile(input.FileName)
		log.Println("Successfully downloaded ", input.FileName)
		response, err := json.Marshal(fileResponse)
		if err != nil {
			log.Println(err.Error())
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string(response), false}, nil
	
	}
	var debitnoteDetails DebitnoteDetails

	var modifiedDate, modifiedBy, invoiceQty, vendorAdd, entityAdd, others, poDate, invoiceDate, poNO, mrinDate, invoiceNo, remarks, itemDescription, vehicleNo, stones sql.NullString

	var rows *sql.Rows
	log.Println("get DebitNote request detail")
	sqlStatement :=
		`SELECT deb.debitnoteid, deb.debitnotedate, deb.debitnoteno,
		deb.vendorid, initcap(v.vendorname) as vendorname, deb.remarks, deb.invoiceno, deb.invoice_qty, s.status,
		deb.entityid, deb.mrinid,deb.mrinno,deb.debitamount,deb.itemid,initcap(i.itemname) as itemname,deb.husk,deb.quality,deb.netrecd,
		deb.moisture, deb.hsccode,deb.others, m.mrindate, m.pono,m.invoicedate, p.podate,
		initcap(v.address1)||','||initcap(v.address2)
		||','||initcap(v.city)||','||v.pincode||','||initcap(v.state)
		||','||'Phone:'||v.phone||','||'Mobile:'||v.mobile||','||'GST NO:'||v.gstin||','||'PAN NO:'||v.panno as vendoraddress,
		e.entityname, initcap(e.address)||','||'GST NO:'||e.gstno as entityaddress, initcap(i.itemdesc) as itemdescription, m.vehicleno, deb.stones
		from dbo.inv_gc_debitnote_master_newpg as deb
        LEFT JOIN dbo.inv_gc_item_master_newpg as i ON deb.itemid = i.itemid
		LEFT JOIN dbo.pur_vendor_master_newpg as v ON deb.vendorid = v.vendorid
        LEFT JOIN dbo.inv_gc_po_mrin_master_newpg as m ON deb.mrinid = m.mrinid
        LEFT JOIN dbo.pur_gc_po_con_master_newpg as p ON m.poid = p.poid
        INNER JOIN dbo.PROJECT_ENTITY_MASTER as e ON deb.entityid = e.entityid
		INNER JOIN dbo.gc_debitnote_status_master_newpg as s ON CAST(deb.status as numeric) = s.id
			where deb.debitnoteid=$1`

	rows, err = db.Query(sqlStatement, input.Debitnoteid)

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&debitnoteDetails.Debitnoteid, &debitnoteDetails.Debitnotedate, &debitnoteDetails.Debitnoteno, &debitnoteDetails.VendorId, &debitnoteDetails.VendorName, &remarks, &invoiceNo,
			&invoiceQty,
			&debitnoteDetails.Status, &debitnoteDetails.EntityId, &debitnoteDetails.Mrinid,
			&debitnoteDetails.MrinNo,
			&debitnoteDetails.DebitAmount, &debitnoteDetails.Itemid, &debitnoteDetails.ItemName, &debitnoteDetails.Husk,
			&debitnoteDetails.Quality, &debitnoteDetails.Netrecd, &debitnoteDetails.Moisture, &debitnoteDetails.HscCode, &others,
			&mrinDate, &poNO, &invoiceDate, &poDate,
			&vendorAdd, &debitnoteDetails.EntityName, &entityAdd, &itemDescription, &vehicleNo, &stones)
	}

	debitnoteDetails.InvoiceQuantity = invoiceQty.String
	debitnoteDetails.VendorAddress = vendorAdd.String
	debitnoteDetails.EntityAddress = entityAdd.String
	debitnoteDetails.Others = others.String
	debitnoteDetails.PoDate = poDate.String
	debitnoteDetails.InvoiceDate = invoiceDate.String
	debitnoteDetails.PoNo = poNO.String
	debitnoteDetails.MrinDate = mrinDate.String
	debitnoteDetails.InvoiceNo = invoiceNo.String
	debitnoteDetails.Remarks = remarks.String
	debitnoteDetails.ItemDescription = itemDescription.String
	debitnoteDetails.VehicleNo = vehicleNo.String
	debitnoteDetails.Stones = stones.String
	//--------------Attached-Debit Note Docs-------------------------//
	log.Println("Fetching Debit Note Docs #")
	sqlStatementDD := `SELECT accounts_docname,vendor_docname
						FROM dbo.pur_gc_debitnote_master_documents_newpg where debitnoteid=$1`
	rowsDD, errDD := db.Query(sqlStatementDD, input.Debitnoteid)
	log.Println("Debit Note Docs Query Executed")
	if errDD != nil {
		log.Println("Debit Note Doc Query failed")
		log.Println(errDD.Error())
	}

	for rowsDD.Next() {
		var dd ReleasedDebitNote
		errDD = rowsDD.Scan(&dd.AccountsDocumentName,&dd.VendorDocumentName)
		debitNoteDocs := append(debitnoteDetails.DebitNoteDocs, dd)
		debitnoteDetails.DebitNoteDocs = debitNoteDocs
		log.Println("added one")

	}

	//---------------------Fetch Audit Log Info-------------------------------------//
	log.Println("Fetching Audit Log Info #")
	sqlStatementAI := `select u.username as createduser, gc.created_date,
	gc.description, v.username as modifieduser, gc.modified_date
   from dbo.auditlog_inv_gc_debitnote_master_newpg gc
   inner join dbo.users_master_newpg u on gc.createdby=u.userid
   left join dbo.users_master_newpg v on gc.modifiedby=v.userid
   where debitnoteid=$1 order by logid desc limit 1`
	rowsAI, errAI := db.Query(sqlStatementAI, input.Debitnoteid)
	log.Println("Audit Info Fetch Query Executed")
	if errAI != nil {
		log.Println("Audit Info Fetch Query failed")
		log.Println(errAI.Error())
	}

	for rowsAI.Next() {
		var al AuditLogGC
		errAI = rowsAI.Scan(&al.CreatedUserName, &al.CreatedDate, &al.Description, &modifiedBy, &modifiedDate)
		al.ModifiedUserName = modifiedBy.String
		al.ModifiedDate = modifiedDate.String
		auditDetails := append(debitnoteDetails.AuditLogDetails, al)
		debitnoteDetails.AuditLogDetails = auditDetails
		log.Println("added one")

	}
	log.Println("Audit Details:", debitnoteDetails.AuditLogDetails)
	res, _ := json.Marshal(debitnoteDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(viewGCDebitNoteDetail)
}
func DownloadFile(fileName string) FileResponse {
	// The session the S3 Uploader will use
	svc := s3.New(session.New())
	
	var fileResponse FileResponse
	fileResponse.FileData = Base64Encoder(svc, Files_Upload_Loc +"/"+fileName)
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