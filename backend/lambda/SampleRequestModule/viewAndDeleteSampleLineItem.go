//CHECKEDIN
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

type SampleLineItemDetails struct {
	SampleCatId        string     `json:"sample_catid"`
	SampleCategory     string     `json:"sample_category"`
	ProductId          string     `json:"product_id"`
	Product            string     `json:"product_type"`
	Description        string     `json:"description"`
	TargetPriceEnabled bool       `json:"targetprice_enabled"`
	TargetPrice        string     `json:"target_price"`
	LineItemId         int        `json:"lineitem_id"`
	AuditLogDetails    []AuditLog `json:"audit_log"`
}

type AuditLog struct {
	CreatedDate      string `json:"created_date"`
	CreatedUserName  string `json:"created_username"`
	ModifiedDate     string `json:"modified_date"`
	ModifiedUserName string `json:"modified_username"`
	Description      string `json:"description"`
	Status           string `json:"status"`
}

type LineItemDetails struct {
	SampleCategory string `json:"sample_category"`
	ProductType    string `json:"product_type"`
	SerialNo       string `json:"serial_no"`
	SampleCode     string `json:"sample_code"`
	PackingType    string `json:"packing_type"`
	DispatchDate   string `json:"dispatch_date"`
	Status         string `json:"status"`
	LineItemId     int    `json:"lineitem_id"`
}

type Input struct {
	Type        string `json:"type"`
	LineItemId  int    `json:"lineitem_id"`
	SampleReqId int    `json:"sample_reqid"`
}

func viewAndDeleteSampleLineItem(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var sampleDetails SampleLineItemDetails

	var rows *sql.Rows
	log.Println("get Sample line item")
	if input.Type == "viewsamplelineitem" {
		sqlStatement :=
			`SELECT 
			sr.productid, initcap(p.productname) as product, sr.description, sr.targetprice, 
            sr.samplecatid, c.sample_category, sr.lineitemid from dbo.cms_sample_request_details sr
			left join dbo.cms_sample_category_master c on c.samplecatid=sr.samplecatid
			left join dbo.prod_product_master p on p.productid=sr.productid
			where sr.lineitemid=$1 limit 1`

		rows, err = db.Query(sqlStatement, input.LineItemId)

		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		defer rows.Close()
		var product, productId, description, targetPrice, samplecatId, sampleCategory sql.NullString
		for rows.Next() {
			err = rows.Scan(&productId, &product, &description, &targetPrice, &samplecatId,
				&sampleCategory, &sampleDetails.LineItemId)
		}
		sampleDetails.ProductId = productId.String
		sampleDetails.Product = product.String
		sampleDetails.Description = description.String
		sampleDetails.TargetPrice = targetPrice.String
		sampleDetails.SampleCatId = samplecatId.String
		sampleDetails.SampleCategory = sampleCategory.String
		if strings.ToUpper(sampleDetails.TargetPrice) != "NO TARGET PRICE" {
			sampleDetails.TargetPriceEnabled = true
		} else {
			sampleDetails.TargetPrice = ""
		}

		//---------------------Fetch Audit Log Info-------------------------------------//
		log.Println("Fetching Audit Log Info #")
		sqlStatementAI := `select u.username as createduser, a.created_date,
			a.description,a.status, v.username as modifieduser, a.modified_date
		   from dbo.auditlog_cms_sample_request_details_newpg a
		   inner join dbo.users_master_newpg u on a.createdby=u.userid
		   left join dbo.users_master_newpg v on a.modifiedby=v.userid
		   where lineitemid=$1 order by logid desc limit 1`
		rowsAI, errAI := db.Query(sqlStatementAI, input.LineItemId)
		log.Println("Audit Info Fetch Query Executed")
		if errAI != nil {
			log.Println("Audit Info Fetch Query failed")
			log.Println(errAI.Error())
		}

		var modifiedBy, modifiedDate sql.NullString

		for rowsAI.Next() {
			var al AuditLog
			errAI = rowsAI.Scan(&al.CreatedUserName, &al.CreatedDate, &al.Description, &al.Status, &modifiedBy, &modifiedDate)
			al.ModifiedUserName = modifiedBy.String
			al.ModifiedDate = modifiedDate.String
			auditDetails := append(sampleDetails.AuditLogDetails, al)
			sampleDetails.AuditLogDetails = auditDetails
			log.Println("added one")

		}
		log.Println("Audit Details:", sampleDetails.AuditLogDetails)

		res, _ := json.Marshal(sampleDetails)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else if input.Type == "deletesamplelineitem" {
		sqlStatement :=
			`delete from dbo.cms_sample_request_details where lineitemid=$1`

		rows, err = db.Query(sqlStatement, input.LineItemId)

		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("Deleted successfully"), false}, nil
	} else if input.Type == "listsamplelineitems" {
		sqlStatement :=
			`SELECT 
			sr.sno,sr.lineitemid, initcap(p.productname) as product, c.sample_category, sr.samplecode,
			date(sr.dispatcheddate) as dispatchdate, sr.packingtype,sr.custsamplestatus
             from dbo.cms_sample_request_details sr
			left join dbo.cms_sample_category_master c on c.samplecatid=sr.samplecatid
			left join dbo.prod_product_master p on p.productid=sr.productid
			where sr.samplereqid=$1 order by sr.sno asc`

		rows, err = db.Query(sqlStatement, input.SampleReqId)

		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		var product, sampleCategory, scode, disdate, pactype, status sql.NullString
		var allLineItemDetails []LineItemDetails
		for rows.Next() {
			var l LineItemDetails
			err = rows.Scan(&l.SerialNo, &l.LineItemId, &product, &sampleCategory,
				&scode, &disdate, &pactype, &status)
			l.ProductType = product.String
			l.SampleCategory = sampleCategory.String
			l.SampleCode = scode.String
			l.DispatchDate = disdate.String
			l.PackingType = pactype.String
			l.Status = status.String
			allLineItemDetails = append(allLineItemDetails, l)
		}

		res, _ := json.Marshal(allLineItemDetails)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
}

func main() {
	lambda.Start(viewAndDeleteSampleLineItem)
}
