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

type GCDetails struct {
	GroupId               string              `json:"group_id"`
	ItemCode              string              `json:"item_code"`
	SCode                 string              `json:"s_code"`
	ItemName              string              `json:"item_name"`
	ItemDesc              string              `json:"item_desc"`
	HsnCode               string              `json:"hsn_code"`
	ConvertionRatio       string              `json:"convertion_ratio"`
	ItemCatId             string              `json:"item_catid"`
	ItemCatName           string              `json:"item_catname"`
	Uom                   string              `json:"uom"`
	UomName               string              `json:"uom_name"`
	DisplayInPo           bool                `json:"display_inpo"`
	DisplayInDailyUpdates bool                `json:"display_in_dailyupdates"`
	IsSpecialCoffee       bool                `json:"is_specialcoffee"`
	CategoryType          string              `json:"cat_type"`
	CoffeeType            string              `json:"coffee_type"`
	LName                 string              `json:"lname"`
	LGroupCode            string              `json:"lgroupcode"`
	AuditLogDetails       []AuditLogGC        `json:"audit_log_gc"`
	StockLocation         []ItemStockLocation `json:"item_stock_location"`
	VendorList            []VendorList        `json:"vendor_list"`
	PoCreatedOnGc         bool                `json:"po_created"`

	//Special composition Info Section---------------------------
	Density       string `json:"density"`
	Moisture      string `json:"moisture"`
	Browns        string `json:"browns"`
	Blacks        string `json:"blacks"`
	BrokenBits    string `json:"broken_bits"`
	InsectedBeans string `json:"insected_beans"`
	Bleached      string `json:"bleached"`
	Husk          string `json:"husk"`
	Sticks        string `json:"sticks"`
	Stones        string `json:"stones"`
	BeansRetained string `json:"beans_retained"`
}

type AuditLogGC struct {
	CreatedDate      string     `json:"created_date"`
	CreatedUserName  string     `json:"created_username"`
	ModifiedDate     NullString `json:"modified_date"`
	ModifiedUserName NullString `json:"modified_username"`
	Description      string     `json:"description"`
}

type ItemStockLocation struct {
	Entity    string `json:"entity"`
	Name      string `json:"name"`
	Quantity  string `json:"quantity"`
	Value     string `json:"value"`
	UnitPrice string `json:"unit_price"`
}

type VendorList struct {
	VendorName  string `json:"vendor_name"`
	ContactName string `json:"contact_name"`
	State       string `json:"state"`
	Country     string `json:"country"`
	City        string `json:"city"`
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

//
type Input struct {
	Itemid string `json:"item_id"`
}

func viewGCDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var gCDetails GCDetails

	var rows *sql.Rows
	log.Println("get GC request detail")
	sqlStatement :=
		`SELECT 
			gc.groupid, gc.itemcode, gc.s_code, gc.itemname, gc.itemdesc,
			gc.hsncode, gc.convertionratio,gc.itemcatid,
			initcap(category.itemcatname) as catname,gc.uom, initcap(u.uomname) as uomname, coffee_type, display_inpo,
			dailyprice_enable, initcap(gc.lname), gc.lgroupcode, gc.cat_type
			from dbo.inv_gc_item_master_newpg gc
			INNER JOIN dbo.INV_ITEM_CATEGORY as category ON gc.itemcatid = category.itemcatid
            INNER JOIN dbo.PROJECT_UOM as u ON gc.uom = u.uom
			where gc.Itemid=$1`

	rows, err = db.Query(sqlStatement, input.Itemid)

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	for rows.Next() {
		err = rows.Scan(&gCDetails.GroupId, &gCDetails.ItemCode, &gCDetails.SCode,
			&gCDetails.ItemName,
			&gCDetails.ItemDesc, &gCDetails.HsnCode, &gCDetails.ConvertionRatio, &gCDetails.ItemCatId,
			&gCDetails.ItemCatName, &gCDetails.Uom, &gCDetails.UomName, &gCDetails.CoffeeType, &gCDetails.DisplayInPo,
			&gCDetails.DisplayInDailyUpdates, &gCDetails.LName, &gCDetails.LGroupCode, &gCDetails.CategoryType)
	}

	if gCDetails.CategoryType == "speciality" {
		gCDetails.IsSpecialCoffee = true
	}

	//Display GC Composition
	log.Println("The GC Composition for the Item ", input.Itemid)
	sqlStatementPOGC1 := `SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, bleached, husk, sticks, stones, beansretained
					FROM dbo.pur_gc_po_composition_master_newpg where itemid=$1`
	rows7, err7 := db.Query(sqlStatementPOGC1, input.Itemid)
	log.Println("GC Fetch Query Executed")
	if err7 != nil {
		log.Println("Fetching GC Composition Details from DB failed")
		log.Println(err7.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	var density, moisture, browns, blacks, brokenBits, insectedBeans, bleached, husk, sticks, stones, beansRetained sql.NullString
	for rows7.Next() {
		err7 = rows7.Scan(&density, &moisture, &browns, &blacks,
			&brokenBits, &insectedBeans, &bleached,
			&husk, &sticks, &stones, &beansRetained)
	}

	gCDetails.Density = density.String
	gCDetails.Moisture = moisture.String
	gCDetails.Browns = browns.String
	gCDetails.Blacks = blacks.String
	gCDetails.BrokenBits = brokenBits.String
	gCDetails.InsectedBeans = insectedBeans.String
	gCDetails.Bleached = bleached.String
	gCDetails.Husk = husk.String
	gCDetails.Sticks = sticks.String
	gCDetails.Stones = stones.String
	gCDetails.BeansRetained = beansRetained.String

	//---------------------Fetch stock location Info-------------------------------------//
	log.Println("Fetching stock location Info #")
	sqlStatementSL := `select initcap(entity.entityname) as entityname, initcap(master.name) as name, loc.quantity,loc.value,loc.unitprice
	from dbo.inv_gc_item_master_newpg gc
	  INNER JOIN dbo.INV_GC_ITEM_LOCATION_OPENING as loc ON gc.lcode = loc.lcode
	  INNER JOIN dbo.inv_gc_location_master as master ON loc.id = master.locationid
	  INNER JOIN dbo.project_entity_master as entity ON master.entityid = entity.entityid
	  where gc.itemid=$1`
	rowsSL, errSL := db.Query(sqlStatementSL, input.Itemid)
	if errSL != nil {
		log.Println("Fetching stock location Info failed")
		log.Println(errSL.Error())
	}

	var entity, name, quantity, value, unitPrice sql.NullString
	for rowsSL.Next() {
		var sl ItemStockLocation
		errSL = rowsSL.Scan(&entity, &name, &quantity, &value, &unitPrice)
		sl.Entity = entity.String
		sl.Name = name.String
		sl.Quantity = quantity.String
		sl.Value = value.String
		sl.UnitPrice = unitPrice.String

		stockDetails := append(gCDetails.StockLocation, sl)
		gCDetails.StockLocation = stockDetails
		log.Println("added one")
	}

	//---------------------Fetch vendor Info Info-------------------------------------//
	log.Println("Fetching vendor info #")
	sqlStatementVL := `SELECT distinct initcap(master.vendorname) as vendorname,initcap(master.contactname) as contactname,initcap(master.state) as state,
	initcap(master.country) as country,initcap(master.city) as city
				from dbo.pur_gc_po_con_master_newpg po
				INNER JOIN dbo.pur_gc_po_details_newpg det ON det.pono = po.pono
				inner join dbo.pur_vendor_master_newpg as master on po.vendorid = master.vendorid
				where det.itemid=$1`
	rowsVL, errVL := db.Query(sqlStatementVL, input.Itemid)
	if errVL != nil {
		log.Println("Fetching vendor Info failed")
		log.Println(errSL.Error())
	}

	for rowsVL.Next() {
		var vl VendorList
		errVL = rowsVL.Scan(&vl.VendorName, &vl.ContactName, &vl.State, &vl.Country, &vl.City)
		vendorDetails := append(gCDetails.VendorList, vl)
		gCDetails.VendorList = vendorDetails
		log.Println("added one")
	}

	//---------------------Fetch already PO exist on GC-------------------------------------//
	log.Println("Fetching vendor info #")
	sqlStatementPc := `select count(*) from dbo.pur_gc_po_details_newpg where itemid=$1`
	rowsPc, errPC := db.Query(sqlStatementPc, input.Itemid)
	if errPC != nil {
		log.Println("Fetching already PO exist on GC Info failed")
		log.Println(errPC.Error())
	}

	var count int
	for rowsPc.Next() {
		errPC = rowsPc.Scan(&count)
	}
	if count > 0 {
		gCDetails.PoCreatedOnGc = true
	}
	if errPC != nil {
		log.Println("Fetching already PO exist on GC failed")
		log.Println(errPC.Error())
	}

	//---------------------Fetch Audit Log Info-------------------------------------//
	log.Println("Fetching Audit Log Info #")
	sqlStatementAI := `select u.username as createduser, gc.created_date,
	gc.description, v.username as modifieduser, gc.modified_date
   from dbo.auditlog_inv_gc_master_newpg gc
   inner join dbo.users_master_newpg u on gc.createdby=u.userid
   left join dbo.users_master_newpg v on gc.modifiedby=v.userid
   where itemid=$1 order by logid desc limit 1`
	rowsAI, errAI := db.Query(sqlStatementAI, input.Itemid)
	log.Println("Audit Info Fetch Query Executed")
	if errAI != nil {
		log.Println("Audit Info Fetch Query failed")
		log.Println(errAI.Error())
	}

	for rowsAI.Next() {
		var al AuditLogGC
		errAI = rowsAI.Scan(&al.CreatedUserName, &al.CreatedDate, &al.Description, &al.ModifiedUserName, &al.ModifiedDate)
		auditDetails := append(gCDetails.AuditLogDetails, al)
		gCDetails.AuditLogDetails = auditDetails
		log.Println("added one")

	}
	log.Println("Audit Details:", gCDetails.AuditLogDetails)

	res, _ := json.Marshal(gCDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(viewGCDetail)
}
