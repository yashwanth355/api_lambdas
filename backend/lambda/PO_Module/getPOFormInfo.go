//Checkedin updated
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
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type SupplierInfo struct {
	Type             string         `json:"type"`
	SupplierName     string         `json:"supplier_name"`
	SupplierID       string         `json:"supplier_id"`
	SupplierType     string         `json:"supplier_type"`
	SupplierCoun     string         `json:"supplier_country"`
	SupplierAddress  string         `json:"supplier_address"`
	SupplierEmail    string         `json:"supplier_email"`
	POCreatedAt      []POCreatedAt  `json:"po_created_at"`
	POCreatedFor     []POCreatedFor `json:"po_created_for"`
	GreenCoffee      []GreenCoffee  `json:"green_coffee_types"`
	SupplierTypeID   string         `json:"supplier_type_id"`
	SupplierTypeName string         `json:"supplier_type_name"`
	ItemID           string         `json:"item_id"`
	POCategory       string         `json:"item_type"`
	POSubCategory    string         `json:"po_subcategory"`
	Tds              string         `json:"tds"`
}
type POCreatedAt struct {
	POTypeID   int    `json:"billing_at_id"`
	POTypeName string `json:"billing_at_name"`
	POAddress  string `json:"billing_at_address"`
}
type POCreatedFor struct {
	POTypeID   int    `json:"delivery_at_id"`
	POTypeName string `json:"delivery_at_name"`
	POAddress  string `json:"delivery_at_address"`
}
type GreenCoffee struct {
	ItemID       string `json:"item_id"`
	ItemName     string `json:"item_name"`
	GCCoffeeType string `json:"gc_type"`
	// GreenCoffee 		[]GreenCoffee `json:"green_coffee_types"`

	Density       string `json:"density"`
	Moisture      string `json:"moisture"`
	Browns        string `json:"browns"`
	Blacks        string `json:"blacks"`
	BrokenBits    string `json:"brokenbits"`
	InsectedBeans string `json:"insectedbeans"`
	Bleached      string `json:"bleached"`
	Husk          string `json:"husk"`
	Sticks        string `json:"sticks"`
	Stones        string `json:"stones"`
	BeansRetained string `json:"beansretained"`
}
type DomesticTaxes struct {
	TaxId   string `json:"tax_id"`
	TaxName string `json:"tax_name"`
}
type MiscCharges struct {
	MiscId      string `json:"misc_id"`
	ChargesName string `json:"misc_charges_name"`
}

func getPOFormInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input SupplierInfo

	err := json.Unmarshal([]byte(request.Body), &input)
	// err2 := json.Unmarshal([]byte(request.Body), &sid)
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

	var res []byte

	var rows *sql.Rows
	if input.Type == "posubcategory" {
		log.Println("get PO Sub Cat:", input.Type)
		sqlStatementPOSub := `select vendortypeid,initcap(vendortypename) from dbo.pur_vendor_types`
		rows, err = db.Query(sqlStatementPOSub)
		log.Println("Query executed")
		var allPOSubs []SupplierInfo
		defer rows.Close()
		for rows.Next() {
			var pos SupplierInfo
			err = rows.Scan(&pos.SupplierTypeID, &pos.SupplierTypeName)
			allPOSubs = append(allPOSubs, pos)
		}

		res, _ = json.Marshal(allPOSubs)

	} else if input.Type == "allsuppliers" {
		log.Println("get vendors", input.Type)
		log.Println("get vendors based on supplierID", input.SupplierTypeID)
		sqlStatementSup := `select vendorid,initcap(vendorname) from dbo.pur_vendor_master_newpg 
		where vendortypeid=$1 and isactive=true and groupid in ('FAC-2','FAC-3','FAC-4','FAC-9','FAC-20','FAC-21')`
		rows, err = db.Query(sqlStatementSup, &input.SupplierTypeID)
		log.Println("Query executed")
		var allSuppliers []SupplierInfo
		defer rows.Close()
		for rows.Next() {
			var a SupplierInfo
			err = rows.Scan(&a.SupplierID, &a.SupplierName)
			allSuppliers = append(allSuppliers, a)
		}

		res, _ = json.Marshal(allSuppliers)

	} else if input.Type == "supplierinfo" {

		log.Println("get vendors", input.Type)
		sqlStatement2 := `select country,vendorid,initcap(vendorname),initcap(address1)||','||initcap(address2)||','||initcap(city)||','||pincode||','||initcap(state)||','||'Phone:'||phone||','||'Mobile:'||mobile||','||'GST NO:'||gstin address,
							email, tds 
							from dbo.pur_vendor_master_newpg where vendorid=$1 and isactive=true`
		rows, err = db.Query(sqlStatement2, input.SupplierID)
		log.Println("Query executed")

		defer rows.Close()

		var tds sql.NullString
		for rows.Next() {

			err = rows.Scan(&input.SupplierCoun, &input.SupplierID, &input.SupplierName, &input.SupplierAddress, &input.SupplierEmail, &tds)
			input.Tds = tds.String
		}

		if input.SupplierCoun != "INDIA" {
			input.SupplierType = "International"
			res, _ = json.Marshal(input)
		} else {
			input.SupplierType = "Domestic"
			res, _ = json.Marshal(input)
		}

	} else if input.Type == "billinginfo" {

		log.Println("get billinginfo details : ", input.Type)
		sqlStatement3 := `select potypeid,initcap(potypename),initcap(potypefullname)||','||initcap(address) as fulladdress from dbo.pur_po_types`
		rows, err = db.Query(sqlStatement3)
		log.Println("Query executed")
		var allPCA []POCreatedAt
		defer rows.Close()
		for rows.Next() {
			var pca POCreatedAt
			err = rows.Scan(&pca.POTypeID, &pca.POTypeName, &pca.POAddress)
			allPCA = append(allPCA, pca)
		}

		res, _ = json.Marshal(allPCA)

	} else if input.Type == "deliveryinfo" {

		log.Println("get deliveryinfo details:", input.Type)
		sqlStatement3 := `select potypeid,initcap(potypename),initcap(potypefullname)||','||initcap(address) as fulladdress from dbo.pur_po_types`
		rows, err = db.Query(sqlStatement3)
		log.Println("Query executed")
		var allPCF []POCreatedFor
		defer rows.Close()
		for rows.Next() {
			var pcf POCreatedFor
			err = rows.Scan(&pcf.POTypeID, &pcf.POTypeName, &pcf.POAddress)
			allPCF = append(allPCF, pcf)
		}

		res, _ = json.Marshal(allPCF)

	} else if input.Type == "greencoffee" {
		var catid string
		if input.POSubCategory == "Domestic" {
			catid = "1002"
		} else {
			catid = "1001"
		}

		log.Println("get GC", input.Type)
		sqlStatement4 := `select itemid,itemname,cat_type from dbo.inv_gc_item_master_newpg where coffee_type=$1 and itemcatid=$2`
		rows, err = db.Query(sqlStatement4, input.POCategory, catid)
		log.Println("GC Query executed")
		var allGc []GreenCoffee
		defer rows.Close()
		for rows.Next() {
			var gc GreenCoffee
			err = rows.Scan(&gc.ItemID, &gc.ItemName, &gc.GCCoffeeType)
			allGc = append(allGc, gc)
		}

		res, _ = json.Marshal(allGc)

	} else if input.Type == "gccomposition" {

		log.Println("get GC new composition based on the GD ID", input.Type)
		log.Println("Entered Item id is:", &input.ItemID)
		sqlStatement5 := `select itemid,density,moisture,browns,blacks,brokenbits,insectedbeans,bleached,husk,sticks,stones,beansretained from dbo.pur_gc_po_composition_master_newpg
							where itemid=$1`
		rows, err = db.Query(sqlStatement5, &input.ItemID)
		log.Println("GC Query executed")
		var allGcComp []GreenCoffee
		defer rows.Close()
		for rows.Next() {
			var gc GreenCoffee
			err = rows.Scan(&gc.ItemID, &gc.Density, &gc.Moisture, &gc.Browns, &gc.Blacks, &gc.BrokenBits, &gc.InsectedBeans,
				&gc.Bleached, &gc.Husk, &gc.Sticks, &gc.Stones, &gc.BeansRetained)
			allGcComp = append(allGcComp, gc)
		}
		log.Println(allGcComp)

		res, _ = json.Marshal(allGcComp)

	} else if input.Type == "getTaxes" {
		log.Println("get Tax types  : ", input.Type)
		sqlStatementGCT := `SELECT taxid, taxname FROM dbo.project_tax_master_newpg where isactive=true`
		rows, err = db.Query(sqlStatementGCT)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		var domTaxes []DomesticTaxes

		defer rows.Close()
		for rows.Next() {
			var dt DomesticTaxes
			err = rows.Scan(&dt.TaxId, &dt.TaxName)
			domTaxes = append(domTaxes, dt)
		}

		res, _ = json.Marshal(domTaxes)

	} else if input.Type == "miscCharges" {
		log.Println("get Tax types  : ", input.Type)
		sqlStatementMC := `SELECT miscid, initcap(name) FROM dbo.project_misc_charges_master_newpg where isactive=true`
		rows, err = db.Query(sqlStatementMC)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		var miscCharges []MiscCharges

		defer rows.Close()
		for rows.Next() {
			var mc MiscCharges
			err = rows.Scan(&mc.MiscId, &mc.ChargesName)
			miscCharges = append(miscCharges, mc)
		}

		res, _ = json.Marshal(miscCharges)

	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}
func main() {
	lambda.Start(getPOFormInfo)
}
