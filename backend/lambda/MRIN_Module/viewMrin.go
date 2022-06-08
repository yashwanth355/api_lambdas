package main

import (
	// "time"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	// "reflect"
	// "strconv"
	// "errors"
	// "database/sql/driver"

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

type MrinRequestDetails struct {
	PoNumber   string `json:"pono"`
	TotalPrice string `json:"totalPrice"`
	// PoId              NullString `json:"poid"`
	InvoiceQuantity   string `json:"invoice_quantity"`
	InvoiceDate       string `json:"invoice_date"`
	ItemID            string `json:"item_id"`
	DeliveryDate      string `json:"delivery_date"`
	DeliveredQuantity string `json:"delivered_quantity"`
	WaymentShortage   string `json:"wayment_shortage"`
	BalanceQuantity   string `json:"balance_quantity"`
	TotalBalQuant     string `json:"total_balance_quantity"`
	PurchasePrice     string `json:"purchase_price"`
	InvoiceNo         string `json:"invoiceno"`
	Status            string `json:"status"`
	QCStatus          string `json:"qcStatus"`
	DetId             string `json:"detid"`
	RelatedDetid      string `json:"related_detid"`
	ExpectedQuantity  string `json:"expected_quantity"`
	CreatedDate       string `json:"createdon"`
	CreatedBy         string `json:"createdby"`
	RequestApproval   string `json:"reqapproval"`
	// VendorId          string `json:"vendor_id"`
	// EntityId          NullString `json:"entityid"`
	Mrinid string `json:"mrinid"`
	// Mrinno            NullString `json:"mrinno"`
	// Mrindate          NullString `json:"mrindate"`
	VehicleNo string `json:"vehicle_no"`
	//New Form changes -Sep5
	WayBillNo   string `json:"wayBillNumber"`
	WayBillDate string `json:"wayBillDate"`
	Locations   string `json:"location"`
	//-------Finance Info
	APStatus           string             `json:"apStatus"`
	APDetails          string             `json:"apDetails"`
	InvoiceAmount      string             `json:"invoiceAmount"`
	ExpectedSpec       []ExpectedGCSpec   `json:"expected_spec"`
	DispatchedSpec     []DispatchedGCSpec `json:"dispatched_spec"`
	DeliveredSpec      []DeliveredGCSpec  `json:"delivered_spec"`
	MiscChargesDetails []MiscCharges      `json:"taxes_misc_charges"`
	MrinValue          string             `json:"mrinvalue"`
}

type Input struct {
	Type         string `json:"type"`
	Mrinid       string `json:"mrinid"`
	FileName     string `json:"file_name"`
	DocKind      string `json:"doc_kind"`
	DocumentName string `json:"document_name"`
	FileContent  string `json:"document_content"`
}
type MiscCharges struct {
	TaxId          string `json:"taxid"`
	TaxName        string `json:"tax_name"`
	TaxPercentage  string `json:"tax_percentage"`
	MiscId         string `json:"misc_id"`
	MiscRate       string `json:"misc_charge_rate"`
	TotalTaxesRate string `json:"total_tax_rate"`
}
type MRINDocumentDetails struct {
	DocumentName string `json:"document_name"`
	FileName     string `json:"file_name"`
	DocKind      string `json:"doc_kind"`
}
type FileResponse struct {
	FileName        string `json:"fileName"`
	FileLink        string `json:"fileLink"`
	FileData        string `json:"fileData"`
	FileContentType string `json:"fileContentType"`
}
type ExpectedGCSpec struct {
	Exp_Density       string `json:"exp_density"`
	Exp_Moisture      string `json:"exp_moisture"`
	Exp_Browns        string `json:"exp_browns"`
	Exp_Blacks        string `json:"exp_blacks"`
	Exp_BrokenBits    string `json:"exp_brokenbits"`
	Exp_InsectedBeans string `json:"exp_insectedbeans"`
	Exp_Bleached      string `json:"exp_bleached"`
	Exp_Husk          string `json:"exp_husk"`
	Exp_Sticks        string `json:"exp_sticks"`
	Exp_Stones        string `json:"exp_stones"`
	Exp_BeansRetained string `json:"exp_beansretained"`
}
type DispatchedGCSpec struct {
	Dis_Density       string `json:"dis_density"`
	Dis_Moisture      string `json:"dis_moisture"`
	Dis_Browns        string `json:"dis_browns"`
	Dis_Blacks        string `json:"dis_blacks"`
	Dis_BrokenBits    string `json:"dis_brokenbits"`
	Dis_InsectedBeans string `json:"dis_insectedbeans"`
	Dis_Bleached      string `json:"dis_bleached"`
	Dis_Husk          string `json:"dis_husk"`
	Dis_Sticks        string `json:"dis_sticks"`
	Dis_Stones        string `json:"dis_stones"`
	Dis_BeansRetained string `json:"dis_beansretained"`
}
type DeliveredGCSpec struct {
	Del_Density       string `json:"del_density"`
	Del_Moisture      string `json:"del_moisture"`
	Del_Browns        string `json:"del_browns"`
	Del_Blacks        string `json:"del_blacks"`
	Del_BrokenBits    string `json:"del_brokenbits"`
	Del_InsectedBeans string `json:"del_insectedbeans"`
	Del_Bleached      string `json:"del_bleached"`
	Del_Husk          string `json:"del_husk"`
	Del_Sticks        string `json:"del_sticks"`
	Del_Stones        string `json:"del_stones"`
	Del_BeansRetained string `json:"del_beansretained"`
}

var Files_Upload_Loc = os.Getenv("S3_MRIN_LOC")

func viewMrin(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	// var rows *sql.Rows
	fmt.Println("Connected!")
	var mrinDetails MrinRequestDetails
	if input.Type == "viewmrin" && input.Mrinid != "" {
		sqlStatementVM1 := `SELECT 
							m.pono,
							m.detid,
							m.related_detid,
							m.expected_quantity,
							m.vehicleno,
		 					m.invoiceno,
		 					m.invoice_quantity, 
							TO_CHAR( m.invoicedate,'DD-MON-YY')as invoicedate,
							TO_CHAR( m.delivery_date,'DD-MON-YY') as delivery_date,
		 					m.delivered_quantity, 
		 					m.wayment_shortage,
		 					m.balance_quantity,
		 					m.waybillno,
							m.waybilldate,
							m.location, 
		 					m.status,
		 					m.ap_status, m.qc_status, m.ap_details, m.invoice_amount,m.mrinvalue,
							po.itemid,
		 					po.total_price,
							po.balance_quantity, 
							po.purchase_price
							from 
							dbo.inv_gc_po_mrin_master_newpg m
							inner join
							dbo.pur_gc_po_details_newpg po
							on po.pono=m.pono
							where 
							m.mrinid=$1`

		log.Println("entered into mrin view query")
		rowsVM1, errVM1 := db.Query(sqlStatementVM1, input.Mrinid)

		if errVM1 != nil {
			log.Println(errVM1)
			return events.APIGatewayProxyResponse{500, headers, nil, errVM1.Error(), false}, nil
		}

		defer rowsVM1.Close()
		var ponum, detid, reldetid, expquan, vehno, invno, invquan, invdate, deldate, delquan, waysh, balquan, waybillno, waybilldate, loc, status,
			apstat, qcstat, apdet, invamt, totprice, pobalquan, itemid, purchasePrice, mrinValue sql.NullString
		for rowsVM1.Next() {
			errVM1 = rowsVM1.Scan(&ponum, &detid, &reldetid, &expquan, &vehno,
				&invno, &invquan, &invdate, &deldate, &delquan,
				&waysh, &balquan, &waybillno, &waybilldate, &loc, &status,
				&apstat, &qcstat, &apdet, &invamt, &mrinValue, &itemid, &totprice, &pobalquan, &purchasePrice)
		}
		mrinDetails.PoNumber = ponum.String
		mrinDetails.DetId = detid.String
		mrinDetails.RelatedDetid = reldetid.String
		mrinDetails.ExpectedQuantity = expquan.String
		mrinDetails.VehicleNo = vehno.String
		mrinDetails.InvoiceNo = invno.String
		mrinDetails.InvoiceQuantity = invquan.String
		mrinDetails.InvoiceDate = invdate.String
		mrinDetails.DeliveryDate = deldate.String
		mrinDetails.DeliveredQuantity = delquan.String
		mrinDetails.WaymentShortage = waysh.String
		mrinDetails.BalanceQuantity = balquan.String
		mrinDetails.WayBillNo = waybillno.String
		mrinDetails.WayBillDate = waybilldate.String
		mrinDetails.Locations = loc.String
		mrinDetails.Status = status.String
		mrinDetails.APStatus = apstat.String
		mrinDetails.QCStatus = qcstat.String
		mrinDetails.APDetails = apdet.String
		mrinDetails.InvoiceAmount = invamt.String
		mrinDetails.TotalPrice = totprice.String
		mrinDetails.TotalBalQuant = pobalquan.String
		mrinDetails.ItemID = itemid.String
		mrinDetails.PurchasePrice = purchasePrice.String
		mrinDetails.MrinValue = mrinValue.String
		if mrinDetails.ItemID != "" && mrinDetails.DetId != "" {
			//---------------------Fetch Expected Dispatch Comp-------------------------------------//
			log.Println("Fetching Expected Dispatch Info #")
			sqlStatementEC := `SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, 
					bleached, husk, sticks, stones, beansretained
						FROM dbo.pur_gc_po_composition_master_newpg 
						where itemid=$1`
			rowsEC, errEC := db.Query(sqlStatementEC, mrinDetails.ItemID)
			log.Println("Audit Info Fetch Query Executed")
			if errEC != nil {
				log.Println("Audit Info Fetch Query failed")
				log.Println(errEC.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, errEC.Error(), false}, nil
			}

			for rowsEC.Next() {
				var edc ExpectedGCSpec
				errEC = rowsEC.Scan(&edc.Exp_Density, &edc.Exp_Moisture, &edc.Exp_Browns, &edc.Exp_Blacks, &edc.Exp_BrokenBits,
					&edc.Exp_InsectedBeans, &edc.Exp_Bleached, &edc.Exp_Husk, &edc.Exp_Sticks, &edc.Exp_Stones,
					&edc.Exp_BeansRetained)
				expCompDetails := append(mrinDetails.ExpectedSpec, edc)
				mrinDetails.ExpectedSpec = expCompDetails
				log.Println("added one")

			}

			//---------------------Fetch Vendor Dispatch Comp-------------------------------------//
			log.Println("Fetching Vendor Dispatch Info #")
			sqlStatementVC := `SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, 
							bleached, husk, sticks, stones, beansretained
							FROM dbo.pur_gc_po_composition_vendor_newpg 
							where detid=$1 and status='Submitted'`
			rowsVC, errVC := db.Query(sqlStatementVC, mrinDetails.DetId)
			log.Println("Audit Info Fetch Query Executed")
			if errVC != nil {
				log.Println("Audit Info Fetch Query failed")
				log.Println(errVC.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, errVC.Error(), false}, nil
			}
			var vdc DispatchedGCSpec
			for rowsVC.Next() {

				errVC = rowsVC.Scan(&vdc.Dis_Density, &vdc.Dis_Moisture, &vdc.Dis_Browns, &vdc.Dis_Blacks, &vdc.Dis_BrokenBits,
					&vdc.Dis_InsectedBeans, &vdc.Dis_Bleached, &vdc.Dis_Husk, &vdc.Dis_Sticks, &vdc.Dis_Stones,
					&vdc.Dis_BeansRetained)
				dispCompDetails := append(mrinDetails.DispatchedSpec, vdc)
				mrinDetails.DispatchedSpec = dispCompDetails
				log.Println("added one")
			}
			//---------------------Fetch Delivered MRIN Dispatch Comp-------------------------------------//
			log.Println("Fetching Delivered MRIN Dispatch GC #")
			sqlStatementQC := `SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, 
							bleached, husk, sticks, stones, beansretained
							FROM dbo.pur_gc_po_composition_vendor_newpg 
							where detid=$1 and status='Delivered'`
			rowsQC, errQC := db.Query(sqlStatementQC, mrinDetails.DetId)
			log.Println("Audit Info Fetch Query Executed")
			if errQC != nil {
				log.Println("Audit Info Fetch Query failed")
				log.Println(errQC.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, errQC.Error(), false}, nil
			}
			var qgc DeliveredGCSpec
			for rowsQC.Next() {
				errQC = rowsQC.Scan(&qgc.Del_Density, &qgc.Del_Moisture, &qgc.Del_Browns, &qgc.Del_Blacks, &qgc.Del_BrokenBits,
					&qgc.Del_InsectedBeans, &qgc.Del_Bleached, &qgc.Del_Husk, &qgc.Del_Sticks, &qgc.Del_Stones,
					&qgc.Del_BeansRetained)
				delCompDetails := append(mrinDetails.DeliveredSpec, qgc)
				mrinDetails.DeliveredSpec = delCompDetails
				log.Println("added one")

			}

			// Get misc tax details
			log.Println("Going to get tax detail for MRIN-", input.Mrinid)
			sqlStatementtaxInfo1 := `select taxid, miscid, perc, taxrate from dbo.INV_GC_MRIN_MISC_DETAILS_TAXES where mrinid=$1`
			rowstax, err := db.Query(sqlStatementtaxInfo1, input.Mrinid)
			if err != nil {
				log.Println("tax Info Fetch Query failed")
				log.Println(err.Error())
			}
			var taxid, perc, miscid, miscrate sql.NullString
			for rowstax.Next() {
				var tx MiscCharges
				err = rowstax.Scan(&taxid, &miscid, &miscrate, &perc)
				tx.TaxId = taxid.String
				tx.TaxPercentage = perc.String
				tx.MiscRate = miscrate.String
				tx.MiscId = miscid.String
				txMiscCharges := append(mrinDetails.MiscChargesDetails, tx)
				mrinDetails.MiscChargesDetails = txMiscCharges
				log.Println("added one misc charge detail", txMiscCharges)
			}

			res, _ := json.Marshal(mrinDetails)
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
		}

	} else if input.Type == "getDocumentsOnMRIN" {
		sqlStatementDoc := `select docname, filename ,dockind from dbo.pur_gc_mrin_master_documents_newpg where mrinid=$1`
		rowsD, errD := db.Query(sqlStatementDoc, input.Mrinid)

		if errD != nil {
			log.Println("Unable to get files uploaded for specific po")
			return events.APIGatewayProxyResponse{500, headers, nil, errD.Error(), false}, nil
		}

		defer rowsD.Close()
		var documents []MRINDocumentDetails
		for rowsD.Next() {
			var dt MRINDocumentDetails
			errD = rowsD.Scan(&dt.DocumentName, &dt.FileName, &dt.DocKind)
			documents = append(documents, dt)
		}

		res, _ := json.Marshal(documents)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else if input.Type == "downloadMRINDocument" {

		log.Println("starting downloaded ", input.FileName)
		fileResponse := DownloadFile(input.FileName)
		log.Println("Successfully downloaded ", input.FileName)
		response, err := json.Marshal(fileResponse)
		if err != nil {
			log.Println(err.Error())
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string(response), false}, nil
	}

	res, _ := json.Marshal(mrinDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}
func main() {
	lambda.Start(viewMrin)
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
