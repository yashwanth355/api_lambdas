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

func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

const venSubmitTemp = `<!DOCTYPE html>
	    <html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		</head>
		<body>
			<h3>Hi,</h3>
			<p>{{.EMessage}}</p>
			<p>Regards,</p>
			<p>{{.EDept}}</p>
		</body>
	</html>`

type Email struct {
	ToEmail string `json:"to_email"`
	ToName  string `json:"name"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type VendorDetails struct {
	Create     bool   `json:"vendor_create"`
	Update     bool   `json:"vendor_update"`
	View       bool   `json:"vendor_view"`
	ListAllPOs bool   `json:"po_list"`
	UserName   string `json:"username"`
	Type       string `json:"type"`
	//
	Status            bool   `json:"status"`
	DispSubmitStatus  bool   `json:"dispatch_sumbit"`
	Vgcompid          string `json:"vgcompid"`
	LastVgIdSno       int    `json:"last_vgidsno"`
	VgIdSno           int    `json:"vgidsno"`
	Detid             string `json:"dispatch_id"`
	RelatedDetid      string `json:"related_detid"`
	InvoiceNo         string `json:"invoice_no"`
	DispatchQuantity  string `json:"dispatch_quantity"`
	DeliveredQuantity string `json:"delivered_quantity"`
	ExpectedQuantity  string `json:"expected_quantity"`
	DispatchDate      string `json:"dispatch_date"`
	CoffeeGrade       string `json:"coffee_grade"`
	VehicleNo         string `json:"vehicle_no"`

	//PO Info Section::
	PoNo       string `json:"po_no"`
	PoDate     string `json:"po_date"`
	PoCategory string `json:"po_category"`

	//Supplier/Vendor Information

	SupplierID    string `json:"supplier_id"`
	SupplierEmail string `json:"supplier_email"`

	//Green Coffee Info Section-Done--------------------------

	ItemID          string         `json:"item_id"`
	Density         string         `json:"density"`
	Moisture        string         `json:"moisture"`
	Browns          string         `json:"browns"`
	Blacks          string         `json:"blacks"`
	BrokenBits      string         `json:"brokenbits"`
	InsectedBeans   string         `json:"insectedbeans"`
	Bleached        string         `json:"bleached"`
	Husk            string         `json:"husk"`
	Sticks          string         `json:"sticks"`
	Stones          string         `json:"stones"`
	BeansRetained   string         `json:"beansretained"`
	AuditLogDetails []AuditLogGCPO `json:"audit_log_gc_po"`
	DispatchForPO   []Dispatches   `json:"dispatches_for_po"`

	//Documents Section
	DocumentsSection []DocumentsUpload `json:"documentsection"`
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

type Dispatches struct {
	DispatchID string `json:"dispatch_id"`
	Dstatus    string `json:"d_status"`
}

type POList struct {
	PONO string `json:"po_no"`
}

type AuditLogGCPO struct {
	CreatedDate    string `json:"createddate"`
	CreatedUserid  string `json:"createduserid"`
	ModifiedDate   string `json:"modifieddate"`
	ModifiedUserid string `json:"modifieduserid"`
	Description    string `json:"description"`
}

var email Email

func insertVendorGCPO(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}

	var v VendorDetails
	var d Dispatches

	// var audit AuditLogGCPO

	err := json.Unmarshal([]byte(request.Body), &v)
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
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
	var rows *sql.Rows
	var res []byte
	if v.Create {
		log.Println("Entered Vendor Info Create Module")

		sqlStatementLR1 := `SELECT vgidsno 
		 					FROM dbo.pur_gc_po_composition_vendor_newpg
							order by vgidsno desc
							LIMIT 1`
		rows, err = db.Query(sqlStatementLR1)
		for rows.Next() {
			err = rows.Scan(&v.LastVgIdSno)
		}
		log.Println("Found existing record: ", v.LastVgIdSno)

		//Generating PO NOs----------------
		v.VgIdSno = v.LastVgIdSno + 1
		v.Vgcompid = "Vendor-" + strconv.Itoa(v.VgIdSno)

		log.Println("Created new idsno: ", v.VgIdSno)
		log.Println("Created Vendor GCID: ", v.Vgcompid)
		// audit.CreatedDate=v.PoDate
		// audit.Description="PO Created"
		// audit.ModifiedDate=v.PoDate
		// audit.ModifiedUserid=v.SupplierEmail

		// po.PoNO = "S4002-" + strconv.Itoa(po.PoIdsNo)
		sqlStatementVGC1 := `INSERT INTO dbo.pur_gc_po_composition_vendor_newpg(
									vgidsno,vgcompid, detid, pono, podate, pocat, invoiceno,
									itemid, dispatch_quan, coffeegrade, vehicle_no, 
									vendorid, email,density, moisture, browns, blacks, brokenbits,
									insectedbeans, bleached, husk, sticks, stones, beansretained,
									status)
									VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`
		_, err := db.Query(sqlStatementVGC1,
			v.VgIdSno,
			v.Vgcompid,
			v.Detid,
			v.PoNo,
			v.PoDate,
			v.PoCategory,
			v.InvoiceNo,
			v.ItemID,
			v.DispatchQuantity,
			v.CoffeeGrade,
			v.VehicleNo,
			v.SupplierID,
			v.SupplierEmail,
			v.Density,
			v.Moisture,
			v.Browns,
			v.Blacks,
			v.BrokenBits,
			v.InsectedBeans,
			v.Bleached,
			v.Husk,
			v.Sticks,
			v.Stones,
			v.BeansRetained,
			"Submitted")
		log.Println("Insert into Vendor GCPO Table Executed")
		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		if v.DocumentsSection != nil && len(v.DocumentsSection) > 0 {
			for i, document := range v.DocumentsSection {
				log.Println("document in loop", i, document)

				sqlStatement1 := `Update dbo.pur_gc_po_master_documents set billofladdingnumber=$1,billofladdingdate=$2,billofentrynumber=$3,billofentrydate=$4,
				invoicenumber=$5, invoicedate=$6, conversationratio=$7 where docid=$8`
				rows, err = db.Query(sqlStatement1, NewNullString(document.Billofladdingnumber), NewNullString(document.Billofladdingdate), NewNullString(document.Billofentrynumber), NewNullString(document.Billofentrydate),
					NewNullString(document.Invoicenumber), NewNullString(document.Invoicedate), NewNullString(document.Conversationratio), document.DocId)
			}
		}
		sub := "Alert: Vendor submitted Green Coffee Specification for PONO: " + v.PoNo
		email.Message = "This is to inform you about the green coffee specification that is submitted by vendor for the dispatch: " + v.Detid
		if findPOEmailid(v.PoNo) != "" {
			smtpSendEmail(venSubmitTemp, sub, email.ToName, email.Message, v.PoNo, "a2z.cclproducts", email.ToEmail)
		}

		// Insert Audit Info
		// log.Println("Entered Audit Module for PO Type")
		// sqlStatementADT := `INSERT INTO dbo.auditlog_pur_gc_master_newpg(
		// 				pono,createdby, created_date, description, modifiedby, modified_date)
		// 				VALUES($1,$2,$3,$4,$5,$6)`
		// _, errADT := db.Query(sqlStatementADT,
		// 						v.PoNo,
		// 						audit.CreatedUserid,
		// 						audit.CreatedDate,
		// 						audit.Description,
		// 						audit.ModifiedUserid,
		// 						audit.ModifiedDate)

		// log.Println("Audit Insert Query Executed")
		// if errADT != nil {
		// 	log.Println("unable to insert Audit Details", errADT)
		// 	}

		// New Query::SELECT po.pono,v.vendorid,v.email,v.vendorname
		// FROM dbo.pur_gc_po_con_master_newpg po
		// inner join dbo.pur_vendor_master v
		// on v.vendorid=po.vendorid
		// where status=1
		// and email='ABC@DEF.COM'

	} else if v.ListAllPOs {
		if v.UserName != "" {
			log.Println("Entered module: get all approved POs")
			sqlStatementLPO1 := `SELECT pom.pono
								FROM dbo.pur_gc_po_con_master_newpg pom
								inner join dbo.pur_vendor_master_newpg ven
								on ven.vendorid=pom.vendorid
								where
								(ven.email=$1)
								and
								(pom.status=3 or pom.status=4)`
			rows, err = db.Query(sqlStatementLPO1, v.UserName)

			var allPOs []POList
			defer rows.Close()
			for rows.Next() {
				var pos POList
				err = rows.Scan(&pos.PONO)
				allPOs = append(allPOs, pos)
			}

			res, _ = json.Marshal(allPOs)
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

		} else {
			return events.APIGatewayProxyResponse{200, headers, nil, string("No Results"), false}, nil
		}
		//

	} else if v.View {
		log.Println("Entered PO View Module")
		if v.PoNo != "" {
			log.Println("Selected PO: ", v.PoNo)
			log.Println("Selected Supplier Email: ", v.UserName)
			sqlStatementVPO := `SELECT pono, podate, pocat, vendorid 
								from dbo.pur_gc_po_con_master_newpg 
								where 
								pono=$1`
			rows, err = db.Query(sqlStatementVPO, v.PoNo)

			defer rows.Close()
			for rows.Next() {
				err = rows.Scan(&v.PoNo, &v.PoDate, &v.PoCategory, &v.SupplierID)
			}

			//Display GC Composition
			log.Println("The GC Composition for the Item #")
			sqlStatementPOGC1 := `SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, bleached, husk, sticks, stones, beansretained
							FROM dbo.pur_gc_po_composition_master_newpg where itemid=$1`
			rows7, err7 := db.Query(sqlStatementPOGC1, v.ItemID)
			log.Println("GC Fetch Query Executed")
			if err7 != nil {
				log.Println("Fetching GC Composition Details from DB failed")
				log.Println(err7.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}

			for rows7.Next() {
				err7 = rows7.Scan(&v.Density, &v.Moisture, &v.Browns, &v.Blacks, &v.BrokenBits, &v.InsectedBeans, &v.Bleached,
					&v.Husk, &v.Sticks, &v.Stones, &v.BeansRetained)

			}
			log.Println("Fetching Density: ", v.Density)
			res, _ = json.Marshal(v)
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
		} else {
			return events.APIGatewayProxyResponse{200, headers, nil, string("No Results"), false}, nil
		}

	} else if v.Type == "dispatch_view" {
		log.Println("View Dispatch Details for DetID#")

		sqlStatementPODV1 := `SELECT 
							quantity,
							dispatch_date,
							parent_detid
		 					FROM dbo.pur_gc_po_dispatch_master_newpg
   							where detid=$1`
		rowsDV1, errDV1 := db.Query(sqlStatementPODV1, v.Detid)
		if errDV1 != nil {
			log.Println("Fetching Dispatch Details from DB failed")
			log.Println(errDV1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errDV1.Error(), false}, nil
		}

		for rowsDV1.Next() {
			errDV1 = rowsDV1.Scan(&v.ExpectedQuantity, &v.DispatchDate, &v.RelatedDetid)

		}
		sqlStatementPODV2 := `SELECT status from 
		 					dbo.pur_gc_po_composition_vendor_newpg
   							where 
							detid=$1`
		rowsDV2, errDV2 := db.Query(sqlStatementPODV2, v.Detid)
		if errDV2 != nil {
			log.Println("Fetching Dispatch Detail Status from Vendor DB failed")
			log.Println(errDV2.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errDV2.Error(), false}, nil
		}

		for rowsDV2.Next() {
			errDV2 = rowsDV2.Scan(&d.Dstatus)

		}
		log.Println("Dispatch Status for PO:", d.Dstatus)
		if d.Dstatus == "Submitted" {
			v.DispSubmitStatus = true
		} else {
			v.DispSubmitStatus = false
		}
		// v.DeliveredQuantity="800"

		res, _ := json.Marshal(v)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else if v.Type == "all_dispatches" {
		log.Println("Fetching All Dispatches")
		sqlStatementAD := `SELECT dis.detid
						 FROM dbo.pur_gc_po_dispatch_master_newpg  dis
						 where dis.pono=$1`
		// left join dbo.pur_gc_po_composition_vendor_newpg ven on ven.detid=dis.detid
		//  and ven.status!='Submitted'
		rowsAD, errAD := db.Query(sqlStatementAD, v.PoNo)
		log.Println("Audit Info Fetch Query Executed")
		if errAD != nil {
			log.Println("Audit Info Fetch Query failed")
			log.Println(errAD.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errAD.Error(), false}, nil
		}

		for rowsAD.Next() {
			var d Dispatches
			errAD = rowsAD.Scan(&d.DispatchID)
			allDispatches := append(v.DispatchForPO, d)
			v.DispatchForPO = allDispatches
			log.Println("added one")
		}
		log.Println("All Dis Details:", v.DispatchForPO)
		res, _ := json.Marshal(v)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(insertVendorGCPO)
}

func smtpSendEmail(temp, subject, name, message, pono, dept, to_email string) (string, error) {
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
		EName    string
		EMessage string
		EDept    string
		PONO     string
	}{
		EName:    name,
		EMessage: message,
		EDept:    dept,
		PONO:     pono,
	})

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to, body.Bytes())
	if err != nil {
		fmt.Println(err)

	}
	return "Email Sent!", nil
}

// Func to find the email id of the user who created the PO from CCL
func findPOEmailid(ponum string) string {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, _ := sql.Open("postgres", psqlInfo)

	defer db.Close()
	log.Println("Function to find po created email id")
	sqlStatementPE := `select um.emailid from
						dbo.pur_gc_po_con_master_newpg po
						inner join 
						dbo.users_master_newpg um
						on po.createdby=um.userid
						where 
						po.pono=$1`
	rowsPE, errPE := db.Query(sqlStatementPE, ponum)
	if errPE != nil {
		log.Println("Error in scanning PO created user email")
		log.Println(errPE.Error())
	}
	log.Println("Scanned PO Created email id")
	for rowsPE.Next() {
		errPE = rowsPE.Scan(&email.ToEmail)
	}
	log.Println("Scanned Email is: ", email.ToEmail)
	return email.ToEmail

}
