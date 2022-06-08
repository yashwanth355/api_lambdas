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

type ApprTop3Pos struct {
	PoNo       string     `json:"po_no"`
	PoDate     string     `json:"po_createddt"`
	VendorId   string     `json:"vendor_id"`
	VendorName string     `json:"vendor_name"`
	GCItem     string     `json:"gcitem_id"`
	Price      NullString `json:"price"`
}

type NullString struct {
	sql.NullString
}

type Input struct {
	Type     string `json:"type"`
	VendorId string `json:"vendor_id"`
	GCItem   string `json:"gcitem_id"`
}

// MarshalJSON for NullString
func (ns *NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

func listTop3ApprovedPos(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var approvedPos []ApprTop3Pos
	var rows *sql.Rows
	if input.Type == "top3apprPosforselectedvendor" {
		sqlStatement := `SELECT q.pono, q.podate,q.vendorid, a.vendorname, det.total_price, det.itemid
		FROM dbo.pur_gc_po_con_master_newpg as q
		INNER JOIN dbo.pur_vendor_master_newpg as a ON q.vendorid = a.vendorid
		INNER JOIN dbo.pur_gc_po_details_newpg det ON det.pono = q.pono
		where (q.status='6' or q.status='5') and (det.itemid=$1 and q.vendorid =$2) order by q.poidsno DESC LIMIT 3`
		rows, err = db.Query(sqlStatement, input.GCItem, input.VendorId)

		defer rows.Close()
		for rows.Next() {
			var po ApprTop3Pos
			err = rows.Scan(&po.PoNo, &po.PoDate, &po.VendorId, &po.VendorName, &po.Price, &po.GCItem)
			approvedPos = append(approvedPos, po)
		}
	} else if input.Type == "top3apprPosforothervendor" {
		sqlStatement := `SELECT q.pono, q.podate,q.vendorid, a.vendorname, det.total_price, det.itemid
		FROM dbo.pur_gc_po_con_master_newpg as q
		INNER JOIN dbo.pur_vendor_master_newpg as a ON q.vendorid = a.vendorid
		INNER JOIN dbo.pur_gc_po_details_newpg det ON det.pono = q.pono
		where (q.status='6' or q.status='5') and (det.itemid=$1 and q.vendorid !=$2) order by q.poidsno DESC LIMIT 3`
		rows, err = db.Query(sqlStatement, input.GCItem, input.VendorId)

		defer rows.Close()
		for rows.Next() {
			var po ApprTop3Pos
			err = rows.Scan(&po.PoNo, &po.PoDate, &po.VendorId, &po.VendorName, &po.Price, &po.GCItem)
			approvedPos = append(approvedPos, po)
		}
	}

	res, _ := json.Marshal(approvedPos)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(listTop3ApprovedPos)
}
