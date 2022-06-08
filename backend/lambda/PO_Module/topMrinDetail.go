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

type TopMrinRecord struct {
	GCItem      string `json:"gcitem_id"`
	MrinDate    string `json:"mrin_date"`
	CGSTPercent string `json:"cgst_per"`
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

type Input struct {
	Type   string `json:"type"`
	Podate string `json:"po_date"`
	GCItem string `json:"gcitem_id"`
}

func topMrinDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var mrinRecord TopMrinRecord
	var rows *sql.Rows
	if input.Type == "topmrinrecord" {
		sqlStatement := `select q.itemid, TO_CHAR(b.mrindate,'yyyy-mm-dd') as mrindate, q.cgst
		from dbo.pur_gc_po_details_taxes_newpg q
	   INNER JOIN dbo.pur_gc_po_con_master_newpg as c ON q.pono = c.pono
	   INNER JOIN dbo.inv_gc_po_mrin_master_newpg as b ON q.pono = b.pono
		where q.itemid=$1 and c.podate <=$2 order by b.mrindate desc limit 1`
		rows, err = db.Query(sqlStatement, input.GCItem, input.Podate)

		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&mrinRecord.GCItem, &mrinRecord.MrinDate, &mrinRecord.CGSTPercent)
		}
	}

	res, _ := json.Marshal(mrinRecord)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(topMrinDetail)
}
