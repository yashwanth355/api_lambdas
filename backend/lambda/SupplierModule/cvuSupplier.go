package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

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
type VendorDetails struct {
	Create         bool   `json:"create"`
	Update         bool   `json:"update"`
	View           bool   `json:"view"`
	CreatedUserID  string `json:"createduserid"`
	ModifiedUserID string `json:"modifieduserid"`
	// CreatedUserName string `json:"createdusername"`
	AutoGenID           string             `json:"autogen_id"`
	VendorId            string             `json:"vendor_id"`
	VendorIdSno         int                `json:"vendor_idsno"`
	VendorName          string             `json:"vendor_name"`
	VendorType          int                `json:"vendor_type"`
	VendorCategory      int                `json:"vendor_cat_name"`
	Lastvendorid        int                `json:"lastvendorid"`
	VendorGroup         string             `json:"vendor_group"`
	VendorTypeid        string             `json:"vendor_type_id"`
	VendorCategoryid    string             `json:"vendor_cat_id"`
	VendorGroupid       string             `json:"vendor_group_id"`
	PanNo               string             `json:"pan_no"`
	GSTIdentificationNo string             `json:"gst_no"`
	MSMESSI             string             `json:"msmessi"`
	BankName            string             `json:"bank_name"`
	Branch              string             `json:"branch"`
	AccountType         string             `json:"account_type"`
	AccountNumber       string             `json:"account_number"`
	IfscCode            string             `json:"ifsc_code"`
	MicrCode            string             `json:"micr_code"`
	ContactName         string             `json:"contact_name"`
	Address1            string             `json:"address1"`
	Address2            string             `json:"address2"`
	City                string             `city:"city"`
	Pincode             string             `json:"pincode"`
	State               string             `json:"state"`
	Country             string             `json:"country"`
	Phone               string             `json:"phone"`
	Mobile              string             `json:"mobile"`
	Email               string             `json:"email"`
	Website             string             `json:"website"`
	AuditLogDetails     []AuditLogSupplier `json:"audit_log_vendor"`
	VendorCategoryName  string             `json:"vendorcatname"`
	VendorGroupName     string             `json:"groupname"`
	IsActive            bool               `json:"isactive"`
	//New fields
	CIN string `json:"cin"`
	TAN string `json:"tan"`
	TDS string `json:"tds"`
}
type AuditLogSupplier struct {
	CreatedDate    string `json:"createddate"`
	CreatedUserid  string `json:"createduserid"`
	ModifiedDate   string `json:"modifieddate"`
	ModifiedUserid string `json:"modifieduserid"`
	Description    string `json:"description"`
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
func NewNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{
		Time:  t,
		Valid: true,
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
func cvuSupplier(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	log.Println("check1")
	var audit AuditLogSupplier
	var vendor VendorDetails
	err := json.Unmarshal([]byte(request.Body), &vendor)
	db, err := sql.Open("postgres", psqlInfo)
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
	var rows *sql.Rows
	if vendor.Create {

		//Find latest vendor id
		sqlStatementVOF1 := `SELECT vendoridsno
							FROM dbo.pur_vendor_master_newpg 
							where vendoridsno is not null
							ORDER BY vendoridsno DESC 
							LIMIT 1`
		rows, err = db.Query(sqlStatementVOF1)

		// var vendor InputAdditionalDetails
		for rows.Next() {
			err = rows.Scan(&vendor.Lastvendorid)
		}
		log.Println("Found exisisting record")
		//Generating vendor NOs
		vendor.VendorIdSno = vendor.Lastvendorid + 1
		vendor.VendorId = "FAC-" + strconv.Itoa(vendor.VendorIdSno)
		vendor.AutoGenID = "Ven-" + strconv.Itoa(vendor.VendorIdSno)
		sqlStatementImp1 := `INSERT INTO dbo.pur_vendor_master_newpg(
			vendorid,
			vendoridsno,
			vendorname,
			vendortypeid,
			vendorcatid, 
			groupid,
			contactname,
			address1,
			address2,
			country,
			state,
			city,
			pincode,
			phone,
			mobile,
			email,
			web,
			panno,
			gstin,
			msme,
			bankname,
			branch,
			accounttype,
			accountno,
			ifscode,
			micrcode,
			isactive,
			cin,
			tan,
			tds,
			auto_gen_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31)`

		_, err = db.Query(sqlStatementImp1,
			vendor.VendorId,
			vendor.VendorIdSno,
			vendor.VendorName,
			NewNullString(vendor.VendorTypeid),
			NewNullString(vendor.VendorCategoryid),
			vendor.VendorGroupid,
			vendor.ContactName,
			vendor.Address1,
			vendor.Address2,
			vendor.Country,
			vendor.State,
			vendor.City,
			vendor.Pincode,
			vendor.Phone,
			vendor.Mobile,
			vendor.Email,
			vendor.Website,
			vendor.PanNo,
			vendor.GSTIdentificationNo,
			vendor.MSMESSI,
			NewNullString(vendor.BankName),
			NewNullString(vendor.Branch),
			NewNullString(vendor.AccountType),
			NewNullString(vendor.AccountNumber),
			NewNullString(vendor.IfscCode),
			NewNullString(vendor.MicrCode),
			vendor.IsActive,
			NewNullString(vendor.CIN),
			NewNullString(vendor.TAN),
			NewNullString(vendor.TDS),
			vendor.AutoGenID)

		log.Println("Insert into vendor Table Executed")

		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		log.Println("Entered vendor details")

		audit.CreatedUserid = vendor.CreatedUserID
		audit.CreatedDate = time.Now().Format("01-02-2006")
		audit.Description = "Vendor created"

		log.Println("Entered Audit Module for supplier Type")
		sqlStatementADT := `INSERT INTO dbo.auditlog_pur_vendor_master_newpg(
						vendorid,createdby, created_date, description)
						VALUES($1,$2,$3,$4)`
		_, errADT := db.Query(sqlStatementADT,
			vendor.VendorId,
			audit.CreatedUserid,
			audit.CreatedDate,
			audit.Description)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}
	} else if vendor.Update && vendor.VendorId != "" {
		log.Println("Entered inside update cond")
		sqlStatementImp1 := `update dbo.pur_vendor_master_newpg
							  set
						  vendorname = $1,
						  vendortypeid = $2,
						  vendorcatid = $3,
						  groupid = $4,
						  contactname = $5,
						  address1 = $6,
						  address2 = $7,
						  country = $8,
						  state = $9,
						  city = $10,
						  pincode = $11,
						  phone = $12,
						  mobile = $13,
						  email = $14,
						  web = $15,
						  panno  =$16,
						  gstin = $17,
						  msme = $18,
						  bankname = $19,
						  branch = $20,
						  accounttype = $21,
						  accountno = $22,
						  ifscode = $23,
						  micrcode = $24,
						  isactive = $25,
						  cin=$26,
						  tan=$27, tds=$28
						  where vendorid = $29`

		rows, err = db.Query(sqlStatementImp1,
			vendor.VendorName,
			NewNullString(vendor.VendorTypeid),
			NewNullString(vendor.VendorCategoryid),
			vendor.VendorGroupid,
			vendor.ContactName,
			vendor.Address1,
			vendor.Address2,
			NewNullString(vendor.Country),
			NewNullString(vendor.State),
			NewNullString(vendor.City),
			vendor.Pincode,
			vendor.Phone,
			NewNullString(vendor.Mobile),
			vendor.Email,
			vendor.Website,
			NewNullString(vendor.PanNo),
			NewNullString(vendor.GSTIdentificationNo),
			NewNullString(vendor.MSMESSI),
			NewNullString(vendor.BankName),
			NewNullString(vendor.Branch),
			NewNullString(vendor.AccountType),
			NewNullString(vendor.AccountNumber),
			NewNullString(vendor.IfscCode),
			NewNullString(vendor.MicrCode),
			vendor.IsActive,
			NewNullString(vendor.CIN),
			NewNullString(vendor.TAN),
			NewNullString(vendor.TDS),
			vendor.VendorId)
		log.Println("just before error nil check")
		if err != nil {
			log.Println("unable to update vendor details", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatementDT1 := `SELECT createdby, created_date
		FROM dbo.auditlog_pur_vendor_master_newpg
		where vendorid=$1 order by logid desc limit 1`

		rows1, _ := db.Query(sqlStatementDT1, vendor.VendorId)
		var createdUser, createdDate sql.NullString

		for rows1.Next() {
			err = rows1.Scan(&createdUser, &createdDate)
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for Vendor")
		sqlStatementADT := `INSERT INTO dbo.auditlog_pur_vendor_master_newpg(
			vendorid,createdby, created_date, description,modifiedby, modified_date)
						VALUES($1,$2,$3,$4,$5,$6)`

		description := "Vendor modified"
		_, errADT := db.Query(sqlStatementADT,
			vendor.VendorId,
			createdUser.String,
			createdDate.String,
			description,
			vendor.ModifiedUserID,
			time.Now().Format("2006-01-02"))

		log.Println("Audit Update Query Executed")
		if errADT != nil {
			log.Println("unable to update Audit Details", errADT)
		}

		//end of update
	} else if vendor.View && vendor.VendorId != "" {

		sqlStatementMDInfo1 := `select d.vendortypeid,d.vendorname, d.groupid, d.contactname,d.address1,d.address2,
							d.city, d.pincode, d.country,d.phone,d.mobile, d.web, d.accountno, d.ifscode,d.bankname,
							d.panno,d.branch, d.gstin,d.msme,d.micrcode,d.accounttype,d.state,m.vendorcatid,m.vendorcatname,n.groupname,d.email,d.isactive,
							d.cin,d.tan,d.tds
							from dbo.pur_vendor_master_newpg d
							inner join dbo.pur_vendor_category as m 
							on d.vendorcatid=m.vendorcatid
							inner join dbo.pur_vendor_groups as n 
							on d.groupid=n.groupid
							where d.vendorid=$1`
		rows1, err1 := db.Query(sqlStatementMDInfo1, vendor.VendorId)
		log.Println("fetch query executed")
		if err1 != nil {
			log.Println("Query failed")
			log.Println(err1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err1.Error(), false}, nil
		}

		var vendortypeid,vendorname, groupid, contactname,address1,address2,
		city, pincode, country,phone,mobile, web, accountno, ifscode,bankname,
		panno,branch, gstin,msme,micrcode,accounttype,state,vendorcatid,vendorcatname,groupname,email,cin,tan,tds sql.NullString
		for rows1.Next() {
			//var al AuditLogSupplier
			err1 = rows1.Scan(&vendortypeid,&vendorname,&groupid,&contactname,&address1,&address2,
				&city, &pincode, &country,&phone,&mobile, &web, &accountno, &ifscode,&bankname,
				&panno,&branch, &gstin,&msme,&micrcode,&accounttype,&state,&vendorcatid,&vendorcatname,&groupname,&email,&vendor.IsActive,&cin,&tan,&tds)
				vendor.VendorTypeid=vendortypeid.String
				vendor.VendorName=vendorname.String
				vendor.VendorGroupid=groupid.String
				vendor.ContactName=contactname.String
				vendor.Address1=address1.String
				vendor.Address2=address2.String
				vendor.City=city.String
				vendor.Pincode=pincode.String
				vendor.Country=country.String
				vendor.Phone= phone.String
				vendor.Mobile= mobile.String
				vendor.Website= web.String
				vendor.AccountNumber= accountno.String
				vendor.IfscCode= ifscode.String
				vendor.BankName= bankname.String
				vendor.PanNo= panno.String
				vendor.Branch= branch.String
				vendor.GSTIdentificationNo= gstin.String
				vendor.MSMESSI= msme.String
				vendor.MicrCode= micrcode.String
				vendor.AccountType= accounttype.String
				vendor.State= state.String
				vendor.VendorCategoryid= vendorcatid.String
				vendor.VendorCategoryName= vendorcatname.String
				vendor.VendorGroupName= groupname.String
				vendor.Email= email.String				
				vendor.CIN= cin.String
				vendor.TAN= tan.String
				vendor.TDS = tds.String
			log.Println("next inside")
		}

		//---------------------Fetch Audit Log Info-------------------------------------//
		log.Println("Fetching Audit Log Info #")
		sqlStatementAI := `select u.username as createduser, a.created_date,
	a.description, v.username as modifieduser, a.modified_date
   from dbo.auditlog_pur_vendor_master_newpg a
   left join dbo.users_master_newpg u on a.createdby=u.userid
   left join dbo.users_master_newpg v on a.modifiedby=v.userid
   where vendorid=$1 order by logid desc limit 1`
		rowsAI, errAI := db.Query(sqlStatementAI, vendor.VendorId)
		log.Println("Audit Info Fetch Query Executed")
		if errAI != nil {
			log.Println("Audit Info Fetch Query failed")
			log.Println(errAI.Error())
		}

		var createdUser, createdDate, modifiedBy, modifiedDate sql.NullString

		for rowsAI.Next() {
			var al AuditLogSupplier
			errAI = rowsAI.Scan(&createdUser, &createdDate, &al.Description, &modifiedBy, &modifiedDate)
			al.CreatedUserid = createdUser.String
			al.CreatedDate = createdDate.String
			al.ModifiedUserid = modifiedBy.String
			al.ModifiedDate = modifiedDate.String
			auditDetails := append(vendor.AuditLogDetails, al)
			vendor.AuditLogDetails = auditDetails
			log.Println("added one")

		}
		log.Println("Audit Details:", vendor.AuditLogDetails)
		res, _ := json.Marshal(vendor)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}

	log.Println("came out")
	return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
}
func main() {
	lambda.Start(cvuSupplier)
}
