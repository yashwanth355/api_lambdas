package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	// "strconv"
	// "time"

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

type VendorDetails struct {
	Create      			bool   `json:"vendor_create"`
	Update      			bool   `json:"vendor_update"`
	View	    			bool   `json:"vendor_view"`
	ListAllPOs				bool   `json:"po_list"`
	UserName				string `json:"username"`
	//
	Status			        bool    `json:"status"`
	Vgcompid				string `json:"vgcompid"`
	LastVgIdSno 			int 	`json:"last_vgidsno"`
	VgIdSno 				int 	`json:"vgidsno"`
	Detid    				string `json:"dispatch_id"`
	InvoiceNo 				string `json:"invoice_no"`
	DispatchQuantity		string `json:"dispatch_quantity"`
	CoffeeGrade				string `json:"coffee_grade"`
	VehicleNo 				string `json:"vehicle_no"`

	//PO Info Section::
	PoNo           	string `json:"po_no"`
	PoDate      	string `json:"po_date"`
	PoCategory    	string `json:"po_category"`	
	
	//Supplier/Vendor Information
	
	SupplierID   	string `json:"supplier_id"`
	SupplierEmail	string `json:"supplier_email"`
	
	//Green Coffee Info Section-Done--------------------------
	
	ItemID 			string `json:"item_id"`
	Density			string `json:"density"` 
	Moisture 		string `json:"moisture"`
	Browns 			string `json:"browns"`
	Blacks 			string `json:"blacks"`
	BrokenBits  	string `json:"brokenbits"`
	InsectedBeans	string `json:"insectedbeans"`
	Bleached 		string `json:"bleached"`
	Husk 			string `json:"husk"`
	Sticks			string `json:"sticks"`
	Stones 			string `json:"stones"`
	BeansRetained	string `json:"beansretained"`

}


func viewVendorGCPO(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var v VendorDetails
	// var audit AuditLogGCPO
	
	err := json.Unmarshal([]byte(request.Body), &v)
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
	var res []byte
	if v.View {
		log.Println("Entered PO View Module")
		
		
		
		//Display GC Composition
			log.Println("The GC Composition for the Item #")
			sqlStatementPOGC1:=`SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, bleached, husk, sticks, stones, beansretained
							FROM dbo.pur_gc_po_composition_master_newpg where itemid=$1`
			rows7, err7 := db.Query(sqlStatementPOGC1,v.ItemID)
			log.Println("GC Fetch Query Executed")
			if err7 != nil {
				log.Println("Fetching GC Composition Details from DB failed")
				log.Println(err7.Error())
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
				}				 
	
			for rows7.Next() {
				err7 = rows7.Scan(&v.Density,&v.Moisture,&v.Browns,&v.Blacks,&v.BrokenBits,&v.InsectedBeans,&v.Bleached,
									&v.Husk,&v.Sticks,&v.Stones,&v.BeansRetained)
		
			}
			log.Println("Fetching Density: ",v.Density)
			res, _ = json.Marshal(v)
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
		
		
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(viewVendorGCPO)
}