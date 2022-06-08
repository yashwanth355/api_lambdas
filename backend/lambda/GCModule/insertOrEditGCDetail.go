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
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type GCDetails struct {
	Update                bool   `json:"update"`
	GroupId               string `json:"group_id"`
	ItemCode              string `json:"item_code"`
	SCode                 string `json:"s_code"`
	ItemName              string `json:"item_name"`
	ItemDesc              string `json:"item_desc"`
	HsnCode               string `json:"hsn_code"`
	ConvertionRatio       string `json:"convertion_ratio"`
	ItemCatId             string `json:"item_catid"`
	Uom                   string `json:"uom"`
	ShowStock             bool   `json:"show_stock"`
	EnableStatus          bool   `json:"enable_status"`
	IsRawMaterial         bool   `json:"is_rawmeterial"`
	DisplayInPo           bool   `json:"display_inpo"`
	DisplayInDailyUpdates bool   `json:"display_in_dailyupdates"`
	IsSpecialCoffee       bool   `json:"is_specialcoffee"`
	CoffeeType            string `json:"coffee_type"`
	CreatedOn             string `json:"created_on"`
	CreatedBy             string `json:"created_by"`
	UpdatedOn             string `json:"updated_on"`
	UpdatedBy             string `json:"updated_by"`
	LCode                 string `json:"lcode"`
	LName                 string `json:"lname"`
	LGroupCode            string `json:"lgroupcode"`
	Itemid                string `json:"item_id"`
	Itemidsno             int    `json:"itemidsno"`

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

func insertOrEditGCDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var gcDetails GCDetails
	err := json.Unmarshal([]byte(request.Body), &gcDetails)
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

	if gcDetails.Update {
		log.Println("Updating GC Item")
		var categoryType string
		if gcDetails.IsSpecialCoffee {
			categoryType = "speciality"
		} else {
			categoryType = "regular"
		}

		sqlStatement1 := `UPDATE dbo.inv_gc_item_master_newpg SET 
		groupid=$1,
		itemcode=$2,
		s_code=$3,
		itemname=$4,
		itemdesc=$5,
		hsncode=$6, 
		convertionratio=$7,
		itemcatid=$8,
		uom=$9,
		coffee_type=$10,
		display_inpo=$11,
		dailyprice_enable=$12,
		updatedon=now(),
		updatedby=$13, lname=$14,lgroupcode=$15, cat_type=$16 where itemid=$17`

		_, err = db.Query(sqlStatement1,
			gcDetails.GroupId, gcDetails.ItemCode, gcDetails.SCode, gcDetails.ItemName,
			gcDetails.ItemDesc, gcDetails.HsnCode, gcDetails.ConvertionRatio, gcDetails.ItemCatId,
			gcDetails.Uom, gcDetails.CoffeeType, gcDetails.DisplayInPo, gcDetails.DisplayInDailyUpdates,
			gcDetails.UpdatedBy, gcDetails.LName, gcDetails.LGroupCode, categoryType, gcDetails.Itemid)

		//Find if item exists in composition master table or not
		log.Println("Find if item exists in composition master table or not")
		sqlStatementFI := `select count(itemid) from dbo.pur_gc_po_composition_master_newpg where itemid=$1`
		rowsFI, errFI := db.Query(sqlStatementFI,gcDetails.Itemid)
		if errFI != nil {
			log.Println(errFI.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errFI.Error(), false}, nil
		}	
		var itemcount int
		for rowsFI.Next() {
			errFI = rowsFI.Scan(&itemcount)
		}
		if itemcount>0{
			sqlStatement2 := `UPDATE dbo.pur_gc_po_composition_master_newpg SET 
			density=$1,
			moisture=$2,
			browns=$3,
			blacks=$4,
	    	brokenbits=$5,
			insectedbeans=$6,
	    	bleached=$7, 
			husk=$8,
			sticks=$9, 
			stones=$10, 
			beansretained=$11 where itemid=$12`

			_, err = db.Query(sqlStatement2,
				gcDetails.Density,
				gcDetails.Moisture,
				gcDetails.Browns,
				gcDetails.Blacks,
				gcDetails.BrokenBits,
				gcDetails.InsectedBeans,
				gcDetails.Bleached,
				gcDetails.Husk,
				gcDetails.Sticks,
				gcDetails.Stones,
				gcDetails.BeansRetained,
				gcDetails.Itemid)

			if err != nil {
				log.Println("unable to update gc details", err.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}

		} else {
			sqlStatementIC := `INSERT INTO dbo.pur_gc_po_composition_master_newpg(
				itemid, density, moisture, browns, blacks, brokenbits, insectedbeans, bleached, husk, sticks, stones, beansretained,createdusername,createddate)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, $10,$11,$12,$13,now());`

			_, errIC := db.Query(sqlStatementIC,
				gcDetails.Itemid,
				gcDetails.Density,
				gcDetails.Moisture,
				gcDetails.Browns,
				gcDetails.Blacks,
				gcDetails.BrokenBits,
				gcDetails.InsectedBeans,
				gcDetails.Bleached,
				gcDetails.Husk,
				gcDetails.Sticks,
				gcDetails.Stones,
				gcDetails.BeansRetained,
				gcDetails.UpdatedBy)

			if errIC != nil {
				log.Println("unable to update gc details", errIC.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, errIC.Error(), false}, nil
			}


		}
			
		

		sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_inv_gc_master_newpg where itemid=$1 order by logid desc limit 1`

		rows1, _ := db.Query(sqlStatementDT1, gcDetails.Itemid)

		for rows1.Next() {
			err = rows1.Scan(&gcDetails.CreatedBy, &gcDetails.CreatedOn)
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for GC")
		sqlStatementADT := `INSERT INTO dbo.auditlog_inv_gc_master_newpg(
						itemid,createdby, created_date, description,modifiedby, modified_date)
						VALUES($1,$2,$3,$4,$5,now())`

		description := "GC Modified"
		_, errADT := db.Query(sqlStatementADT,
			gcDetails.Itemid,
			gcDetails.CreatedBy,
			gcDetails.CreatedOn,
			description,
			gcDetails.UpdatedBy)

		log.Println("Audit Update Query Executed")
		if errADT != nil {
			log.Println("unable to update Audit Details", errADT)
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("Updated successfully"), false}, nil
	} else {

		sqlStatementDT1 := `select itemidsno from dbo.inv_gc_item_master_newpg order by itemidsno desc limit 1`

		rows1, err := db.Query(sqlStatementDT1)

		for rows1.Next() {
			err = rows1.Scan(&gcDetails.Itemidsno)
		}

		gcDetails.Itemidsno = gcDetails.Itemidsno + 1
		gcDetails.Itemid = "FAC-" + strconv.Itoa(gcDetails.Itemidsno)
		gcDetails.LCode = "GCITEM-FAC-" + strconv.Itoa(gcDetails.Itemidsno)
		gcDetails.LGroupCode = "GCITEM-" + gcDetails.GroupId

		var categoryType string
		if gcDetails.IsSpecialCoffee {
			categoryType = "speciality"
		} else {
			categoryType = "regular"
		}

		log.Println("Itemidsno", gcDetails.Itemidsno, "Itemid", gcDetails.Itemid, "LCode", gcDetails.LCode, "LGroupCode", gcDetails.LGroupCode)

		sqlStatement2 := `INSERT INTO dbo.inv_gc_item_master_newpg (
			groupid,
			itemcode,
			s_code,
			itemname,
			itemdesc,
			hsncode, 
			convertionratio,
			itemcatid,
			uom,
			coffee_type,
			display_inpo,
			dailyprice_enable,
			createdby, lcode,lname,lgroupcode, itemid, itemidsno, cat_type,createdon) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,$15,$16,$17,$18,$19,now())`
		_, err = db.Query(sqlStatement2,
			gcDetails.GroupId,
			gcDetails.ItemCode,
			gcDetails.SCode,
			gcDetails.ItemName,
			gcDetails.ItemDesc,
			gcDetails.HsnCode,
			gcDetails.ConvertionRatio,
			gcDetails.ItemCatId,
			gcDetails.Uom,
			gcDetails.CoffeeType,
			gcDetails.DisplayInPo,
			gcDetails.DisplayInDailyUpdates,
			gcDetails.CreatedBy, gcDetails.LCode, gcDetails.LName, gcDetails.LGroupCode, gcDetails.Itemid, gcDetails.Itemidsno, categoryType)

		if err != nil {
			log.Println("Insert to GCDetails request failed", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatement3 := `INSERT INTO dbo.pur_gc_po_composition_master_newpg (density,
			 moisture, browns, blacks,
			 brokenbits, insectedbeans,
		     bleached, husk, sticks, stones, beansretained, itemid) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

		_, err = db.Query(sqlStatement3,
			gcDetails.Density,
			gcDetails.Moisture,
			gcDetails.Browns,
			gcDetails.Blacks,
			gcDetails.BrokenBits,
			gcDetails.InsectedBeans,
			gcDetails.Bleached,
			gcDetails.Husk,
			gcDetails.Sticks,
			gcDetails.Stones,
			gcDetails.BeansRetained,
			gcDetails.Itemid)

		if err != nil {
			log.Println("Insert to GC composition data", err.Error())
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for GC")
		sqlStatementADT := `INSERT INTO dbo.auditlog_inv_gc_master_newpg(
						itemid,createdby, description,created_date)
						VALUES($1,$2,$3,now())`

		description := "GC Created"
		_, errADT := db.Query(sqlStatementADT,
			gcDetails.Itemid,
			gcDetails.CreatedBy,
			description)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("Created successfully"), false}, nil
	}
}

func main() {
	lambda.Start(insertOrEditGCDetail)
}
