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

type MrinInfoList struct {
	MrinId             string `json:"mrin_id"`
	MrinNo             string `json:"mrin_no"`
	InvoiceQuantity    string `json:"invoice_quantity"`
	InvoiceNo          string `json:"invoice_no"`
	GcItem             string `json:"gc_itemname"`
	ItemId             string `json:"item_id"`
	Husk               string `json:"husk"`
	Moisture           string `json:"moisture"`
	Stones             string `json:"stones"`
	HsnCode            string `json:"hsn_code"`
	DispatchedQuantity string `json:"net_weightrecorded"`
}

type VendorInfoList struct {
	VendorId   string `json:"vendor_id"`
	VendorName string `json:"vendor_name"`
}

type Input struct {
	Type          string `json:"type"`
	EntityId      string `json:"entity_id"`
	VendorId      string `json:"vendor_id"`
	DebitNoteDate string `json:"debit_notedate"`
}

func getGCDebitNoteCreationInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	var res []byte
	var rows *sql.Rows
	if input.Type == "mrinsOnVendor" && input.EntityId !="" {
		log.Println("get all mrins", input.Type)
		var invoiceNo, invoiceQuantity, itemId, itemName, hsncode, moisture, husk, dispatchQuantity, stones sql.NullString
		sqlStatement := `select m.mrinid, m.mrinno, m.invoiceno, m.invoice_quantity,p.itemid, initcap(i.itemname) as gcitem,
		i.hsncode, c.moisture, c.husk, c.dispatch_quan, c.stones
				from dbo.inv_gc_po_mrin_master_newpg m
			   Left JOIN dbo.pur_gc_po_con_master_newpg as p ON p.poid = m.poid
			   Left JOIN dbo.inv_gc_item_master_newpg as i ON p.itemid = i.itemid
			   Left JOIN dbo.pur_gc_po_composition_vendor_newpg as c ON m.detid = c.detid
			   where m.entityid=$1 and m.vendorid=$2 
			   and m.mrindate <=$3 and m.status='Pending with Finance' and c.status='Delivered'`
		rows, err = db.Query(sqlStatement, input.EntityId, input.VendorId, input.DebitNoteDate)

		var allMrinInfoList []MrinInfoList
		defer rows.Close()
		for rows.Next() {
			var ct MrinInfoList
			err = rows.Scan(&ct.MrinId, &ct.MrinNo, &invoiceNo, &invoiceQuantity, &itemId, &itemName,
				&hsncode, &moisture, &husk, &dispatchQuantity, &stones)
			ct.InvoiceNo = invoiceNo.String
			ct.InvoiceQuantity = invoiceQuantity.String
			ct.ItemId = itemId.String
			ct.GcItem = itemName.String
			ct.HsnCode = hsncode.String
			ct.Moisture = moisture.String
			ct.Husk = husk.String
			ct.DispatchedQuantity = dispatchQuantity.String
			ct.Stones = stones.String

			allMrinInfoList = append(allMrinInfoList, ct)
		}

		res, _ = json.Marshal(allMrinInfoList)
	} else if input.Type == "allVendors" {
		log.Println("get all vendors", input.Type)
		sqlStatement := `select vendorid, initcap(vendorname) as vendorname from dbo.pur_vendor_master_newpg order by vendoridsno desc`
		rows, err = db.Query(sqlStatement)

		var allVendors []VendorInfoList
		defer rows.Close()
		for rows.Next() {
			var ct VendorInfoList
			err = rows.Scan(&ct.VendorId, &ct.VendorName)
			allVendors = append(allVendors, ct)
		}

		res, _ = json.Marshal(allVendors)
	} else {
		return events.APIGatewayProxyResponse{500, headers, nil, string("Invalid input selected"), false}, nil
	

	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getGCDebitNoteCreationInfo)
}
