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

type MrinDetails struct {
	Type         string `json:"type"`
	ItemID       string `json:"itemid"`
	Role         string `json:"role"`
	CreateMRIN   bool   `json:"createmrin"`
	Updatespec   bool   `json:"update_spec"`
	Update       bool   `json:"update"`
	PoId         string `json:"poid"`
	PoNO         string `json:"pono"`
	Detid        string `json:"detid"`
	RelatedDetid string `json:"related_detid"`

	CreatedDate string `json:"createdon"`
	CreatedBy   string `json:"createdby"`

	VendorId      string `json:"vendor_id"`
	EntityId      string `json:"entityid"`
	LastMrinidsno int    `json:"last_mrinidsno"`
	Mrinid        string `json:"mrinid"`
	Mrinidsno     int    `json:"mrinidsno"`
	LastVgidsno   int    `json:"last_vgidsno"`
	Vgidsno       int    `json:"vgidsno"`
	Vgcompid      string `json:"vgcompid"`
	Mrinno        string `json:"mrinno"`
	Mrindate      string `json:"mrindate"`
	LastDetId     int    `json:"last_detid"`
	Status        string `json:"status"`

	// DeliveredSpec []DeliveredGCSpec`json:"delivered_spec"`
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
func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func updateMRINGCSpec(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var sd MrinDetails
	// var dgc DeliveredGCSpec
	err := json.Unmarshal([]byte(request.Body), &sd)
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

	if (sd.Detid != "") && (sd.Updatespec) {
		log.Println("Entered")
		log.Println("Checking if a record with vgcompid exists in DB")
		sqlStatementQCF1 := `select vgcompid from dbo.pur_gc_po_composition_vendor_newpg
							where detid=$1 and status = 'Delivered'`
		rowsQCF1, errQCF1 := db.Query(sqlStatementQCF1, sd.Detid)
		if errQCF1 != nil {
			log.Println("Finding Vgcompid in Vendor GC Spec DB failed")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		for rowsQCF1.Next() {
			errQCF1 = rowsQCF1.Scan(&sd.Vgcompid)
		}
		if sd.Vgcompid == "" {
			log.Println("Found no record with vgcompid in DB,Inserting a new record")
			// Insert new GC Spec record in comp table
			sqlStatementQCF2 := `SELECT vgidsno 
								FROM dbo.pur_gc_po_composition_vendor_newpg 
								where vgidsno is not null
								ORDER BY vgidsno DESC 
								LIMIT 1`
			rowsQCF2, errQCF2 := db.Query(sqlStatementQCF2)
			if errQCF2 != nil {
				log.Println("Finding vgidsno failed")
				return events.APIGatewayProxyResponse{500, headers, nil, errQCF2.Error(), false}, nil
			}
			for rowsQCF2.Next() {
				errQCF2 = rowsQCF2.Scan(&sd.LastVgidsno)
			}
			sd.Vgidsno = sd.LastVgidsno + 1
			sd.Vgcompid = "Mrin-GC-Spec-" + strconv.Itoa(sd.Vgidsno)
			sd.Status = "Delivered"

			sqlStatementQCF3 := `INSERT INTO dbo.pur_gc_po_composition_vendor_newpg(
								vgcompid, 
								detid, 
								pono, 
								itemid, 
								density, moisture, browns, 
								blacks, brokenbits, insectedbeans, 
								bleached, husk, sticks, stones, beansretained, 
								status,vgidsno)
								VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17);`
			_, errQCF3 := db.Query(sqlStatementQCF3,
				sd.Vgcompid,
				sd.Detid,
				sd.PoNO,
				sd.ItemID,
				NewNullString(sd.Del_Density),
				NewNullString(sd.Del_Moisture),
				NewNullString(sd.Del_Browns),
				NewNullString(sd.Del_Blacks),
				NewNullString(sd.Del_BrokenBits),
				NewNullString(sd.Del_InsectedBeans),
				NewNullString(sd.Del_Bleached),
				NewNullString(sd.Del_Husk),
				NewNullString(sd.Del_Sticks),
				NewNullString(sd.Del_Stones),
				NewNullString(sd.Del_BeansRetained),
				sd.Status,
				sd.Vgidsno)

			if errQCF3 != nil {
				log.Println("Insert of delivered gc spec failed")
				return events.APIGatewayProxyResponse{500, headers, nil, errQCF3.Error(), false}, nil
			}
			sqlStatementQCF4 := `UPDATE dbo.pur_gc_po_composition_vendor_newpg
								SET 
								pono=subquery.pono,
								podate=subquery.podate,
								pocat=subquery.pocat,
								invoiceno=subquery.invoiceno,
								dispatch_quan=subquery.dispatch_quan,
								coffeegrade=subquery.coffeegrade,
								vehicle_no=subquery.vehicle_no,
								vendorid=subquery.vendorid,
								email=subquery.email,
								mrinid=subquery.mrinid
								 FROM (SELECT pono, podate, pocat, invoiceno,dispatch_quan, coffeegrade, vehicle_no, vendorid, email,mrinid,detid	FROM dbo.pur_gc_po_composition_vendor_newpg where status='Submitted') AS subquery
								WHERE dbo.pur_gc_po_composition_vendor_newpg.detid=$1 and dbo.pur_gc_po_composition_vendor_newpg.status='Delivered'`

			_, errQCF4 := db.Query(sqlStatementQCF4, sd.Detid)
			if errQCF4 != nil {
				log.Println("Updating composition details for both delivered and submitted failed")
				// return events.APIGatewayProxyResponse{500, headers, nil, errQCF4.Error(), false}, nil
			}

		} else {
			log.Println("Found an existing record with vgcompid in DB,hence updating existing record")
			// Update GC spec in comp table
			sqlStatementQU := `UPDATE dbo.pur_gc_po_composition_vendor_newpg
								SET 
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
								beansretained=$11
								where 
								detid=$12
								and
								status='Delivered'`

			_, err = db.Query(sqlStatementQU,
				sd.Del_Density,
				sd.Del_Moisture,
				sd.Del_Browns,
				sd.Del_Blacks,
				sd.Del_BrokenBits,
				sd.Del_InsectedBeans,
				sd.Del_Bleached,
				sd.Del_Husk,
				sd.Del_Sticks,
				sd.Del_Stones,
				sd.Del_BeansRetained,
				sd.Detid)

			if err != nil {
				log.Println("unable to update gc details", err.Error())
			}

		}
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil

}
func main() {
	lambda.Start(updateMRINGCSpec)
}
