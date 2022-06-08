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

type QuoteItemDetails struct {
	QuoteId                  int        `json:"quote_id"`
	QuoteNumber              string     `json:"quote_number"`
	QuoteLineItemNumber      string     `json:"quotelineitem_number"`
	SampleId                 string     `json:"sample_id"`
	SampleCode               string     `json:"sample_code"`
	PackingId                int        `json:"category_id"`
	CategoryName             string     `json:"category_name"`
	PackingTypeId            int        `json:"categorytype_id"`
	CategoryTypeName         string     `json:"categorytype_name"`
	WeightId                 int        `json:"weight_id"`
	WeightName               string     `json:"weight_name"`
	CartonTypeId             int        `json:"cartontype_id"`
	CartonTypeName           string     `json:"cartontype_name"`
	CapTypeId                int        `json:"captype_id"`
	CapTypeName              string     `json:"captype_name"`
	SecondaryId              int        `json:"secondary_id"`
	SecondaryName            string     `json:"secondary_name"`
	NoOfSecondaryId          int        `json:"noofsecondary_id"`
	NoOfSecondaryName        string     `json:"noofsecondary_name"`
	UPCId                    int        `json:"upc_id"`
	UPCName                  string     `json:"upc_name"`
	Palletizationrequireid   int        `json:"palletizationrequire_id"`
	CustomerBrandName        string     `json:"customerbrand_name"`
	AdditionalRequirements   string     `json:"additional_req"`
	ExpectedOrder            int        `json:"expectedorder_kgs"`
	IsReqNewPacking          int        `json:"isreqnew_packing"`
	CreatedDate              string     `json:"created_date"`
	CreatedBy                string     `json:"created_by"`
	CoffeeType               string     `json:"coffee_type"`
	InCoterms                string     `json:"incoterms"`
	Destination              string     `json:"destination"`
	NewPackingDescription    string     `json:"taskdesc"`
	NewPackTypeRequestStatus string     `json:"new_packtype_status"`
	AuditLogDetails          []AuditLog `json:"audit_log"`

	BasePrice             string `json:"baseprice"`
	Margin                string `json:"margin"`
	MarginPercentage      string `json:"margin_percentage"`
	FinalPrice            string `json:"final_price"`
	NegativeMarginStatus  string `json:"negativemargin_status"`
	NegativeMarginRemarks string `json:"negativemargin_remarks"`
	NegativeMarginReason  string `json:"negativemargin_reason"`
	//
	CustApprove              string `json:"customer_approval"`
	GmsApprovalStatus        string `json:"gms_approvalstatus"`
	GmsRejectionRemarks      string `json:"gms_rejectionremarks"`
	CustomerRejectionRemarks string `json:"customer_rejectionremarks"`
	ConfirmedOrderQuantity   string `json:"confirmed_orderquantity"`
}

type AuditLog struct {
	CreatedDate      string `json:"created_date"`
	CreatedUserName  string `json:"created_username"`
	ModifiedDate     string `json:"modified_date"`
	ModifiedUserName string `json:"modified_username"`
	Description      string `json:"description"`
	Status           string `json:"status"`
}

type PackingCategory struct {
	PackingId           int    `json:"category_id"`
	PackingCategoryName string `json:"category_name"`
}

type PackingCategoryType struct {
	PackingTypeId           int    `json:"categorytype_id"`
	PackingCategoryTypeName string `json:"categorytype_name"`
}

type SecondaryPacking struct {
	SecondaryId   int    `json:"secondary_id"`
	SecondaryName string `json:"secondary_name"`
}

type NoOfSecondarys struct {
	NoOfSecondaryId   int    `json:"noofsecondary_id"`
	NoOfSecondaryName string `json:"noofsecondary_name"`
}

type UPC struct {
	UPCId   int    `json:"upc_id"`
	UPCName string `json:"upc_name"`
}

type CartonType struct {
	CartonTypeId   int    `json:"cartontype_id"`
	CartonTypeName string `json:"cartontype_name"`
}

type CapType struct {
	CapTypeId   int    `json:"captype_id"`
	CapTypeName string `json:"captype_name"`
}

type PackingWeight struct {
	WeightId   int    `json:"weight_id"`
	WeightName string `json:"weight_name"`
}

type Sample struct {
	SampleCode string `json:"sample_code"`
	CoffeeType string `json:"coffee_tyoe"`
	SampleId   string `json:"sample_id"`
}

type Input struct {
	Type            string `json:"type"`
	AccountId       string `json:"account_id"`
	PackingId       int    `json:"category_id"`
	PackingTypeId   int    `json:"categorytype_id"`
	WeightId        int    `json:"weight_id"`
	SecondaryId     int    `json:"secondary_id"`
	QuoteLineItemId int    `json:"quotelineitem_id"`
	CreatedBy       string `json:"created_by"`
}

func getQuoteLineCreationInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	if input.Type == "allCategories" {
		log.Println("get packing categories", input.Type)
		sqlStatement := `select categoryid, categoryname from dbo.cms_prod_pack_category`
		rows, err = db.Query(sqlStatement)

		var allPackingCategories []PackingCategory
		defer rows.Close()
		for rows.Next() {
			var category PackingCategory
			err = rows.Scan(&category.PackingId, &category.PackingCategoryName)
			allPackingCategories = append(allPackingCategories, category)
		}

		res, _ = json.Marshal(allPackingCategories)
	} else if input.Type == "categoryTypes" {
		log.Println("get packing categories", input.Type)
		sqlStatement := `select categorytypeid, categorytypename from dbo.cms_prod_pack_category_type where categoryid=$1`
		rows, err = db.Query(sqlStatement, input.PackingId)

		var allPackingCategoryTypes []PackingCategoryType
		defer rows.Close()
		for rows.Next() {
			var categoryType PackingCategoryType
			err = rows.Scan(&categoryType.PackingTypeId, &categoryType.PackingCategoryTypeName)
			allPackingCategoryTypes = append(allPackingCategoryTypes, categoryType)
		}

		res, _ = json.Marshal(allPackingCategoryTypes)
	} else if input.Type == "weights" {
		log.Println("get weights", input.Type)
		sqlStatement := `select weightid, weightname from dbo.cms_prod_pack_category_weight where categoryid=$1 and categorytypeid=$2`
		rows, err = db.Query(sqlStatement, input.PackingId, input.PackingTypeId)

		var allPackingWeights []PackingWeight
		defer rows.Close()
		for rows.Next() {
			var weight PackingWeight
			err = rows.Scan(&weight.WeightId, &weight.WeightName)
			allPackingWeights = append(allPackingWeights, weight)
		}

		res, _ = json.Marshal(allPackingWeights)
	} else if input.Type == "secondarypackings" {
		log.Println("get secondary packings", input.Type)
		sqlStatement := `select packsecondaryid, packsecondaryname from dbo.cms_prod_pack_packsecondary 
		where categoryid=$1 and categorytypeid=$2 and weightid=$3`
		rows, err = db.Query(sqlStatement, input.PackingId, input.PackingTypeId, input.WeightId)

		var allSecondaryPackings []SecondaryPacking
		defer rows.Close()
		for rows.Next() {
			var pack SecondaryPacking
			err = rows.Scan(&pack.SecondaryId, &pack.SecondaryName)
			allSecondaryPackings = append(allSecondaryPackings, pack)
		}

		res, _ = json.Marshal(allSecondaryPackings)
	} else if input.Type == "noofsecondarypacks" {
		log.Println("get no of secondary packings", input.Type)
		sqlStatement := `select noofsecondaryid, noofsecondaryname from dbo.cms_prod_pack_noofsecondary 
		where categoryid=$1 and categorytypeid=$2 and weightid=$3 and packsecondaryid=$4`
		rows, err = db.Query(sqlStatement, input.PackingId, input.PackingTypeId, input.WeightId, input.SecondaryId)

		var allNoOfSecondarys []NoOfSecondarys
		defer rows.Close()
		for rows.Next() {
			var secondaryPack NoOfSecondarys
			err = rows.Scan(&secondaryPack.NoOfSecondaryId, &secondaryPack.NoOfSecondaryName)
			allNoOfSecondarys = append(allNoOfSecondarys, secondaryPack)
		}

		res, _ = json.Marshal(allNoOfSecondarys)
	} else if input.Type == "samplecode" {
		log.Println("get sample codes", input.Type)
		sqlStatement := `select samplecode, initcap(p.productname) as coffeetype, sampleid
		from dbo.qua_product_sample_master h
			   INNER JOIN dbo.prod_product_master p on p.productid = h.productid
			   INNER JOIN dbo.sales_customer_master s on s.ref_custid = h.custid`

		//where s.crmid=$1

		rows, err = db.Query(sqlStatement)

		var allSamples []Sample
		defer rows.Close()
		for rows.Next() {
			var code Sample
			err = rows.Scan(&code.SampleCode, &code.CoffeeType, &code.SampleId)
			allSamples = append(allSamples, code)
		}

		res, _ = json.Marshal(allSamples)
	} else if input.Type == "upcs" {
		log.Println("get upcs", input.Type)
		sqlStatement := `select upcid, upcname from dbo.cms_prod_pack_upc 
		where categoryid=$1 and categorytypeid=$2 and weightid=$3 and packsecondaryid=$4`
		rows, err = db.Query(sqlStatement, input.PackingId, input.PackingTypeId, input.WeightId, input.SecondaryId)

		var allUPCs []UPC
		defer rows.Close()
		for rows.Next() {
			var upc UPC
			err = rows.Scan(&upc.UPCId, &upc.UPCName)
			allUPCs = append(allUPCs, upc)
		}

		res, _ = json.Marshal(allUPCs)
	} else if input.Type == "cartontypes" {
		log.Println("get carton types", input.Type)
		sqlStatement := `select cartontypeid, cartontypename from dbo.cms_prod_pack_cartontype`
		rows, err = db.Query(sqlStatement)

		var allCartonTypes []CartonType
		defer rows.Close()
		for rows.Next() {
			var carton CartonType
			err = rows.Scan(&carton.CartonTypeId, &carton.CartonTypeName)
			allCartonTypes = append(allCartonTypes, carton)
		}

		res, _ = json.Marshal(allCartonTypes)
	} else if input.Type == "captypes" {
		log.Println("get cap types", input.Type)
		sqlStatement := `select captypeid, captypename from dbo.cms_prod_pack_captype`
		rows, err = db.Query(sqlStatement)

		var allCapTypes []CapType
		defer rows.Close()
		for rows.Next() {
			var cap CapType
			err = rows.Scan(&cap.CapTypeId, &cap.CapTypeName)
			allCapTypes = append(allCapTypes, cap)
		}

		res, _ = json.Marshal(allCapTypes)
	} else if input.Type == "Viewquotelineitem" {
		log.Println("get view quote line item", input.Type)
		var newPackTypeRequestStatus, packingId, customerBrandName, packDescription, additionalRequirement, destination, packCategoryTypeId, packWeightId, packupcId, packCartonId, packCapTypeId,
			packSecondaryId, packNoofSecondaryId, isReqNewPack, productName, parllelizationRequired,
			margin, marginPercenatge, basePrice, finalPrice, custApprove, gmsApprovalStatus, rejectionRemarks, gmsRejectionComments, confirmedOrderQuantity, nmstatus, nmremarks, nmreason sql.NullString
		sqlStatement := `select q.quoteid,w.quotenumber, q.lineitemnumber,
		q.sampleid,
		h.samplecode,
		q.expectedorderqty,
		q.packcategoryid,
		q.packcategorytypeid,
		q.packweightid,
		q.packupcid,
		q.packcartontypeid,
		q.packcaptypeid,
		q.packsecondaryid,
		q.packnoofsecondaryid,
		q.palletizationrequireid,
		q.customerbrandname,
		q.additionalrequirements,
		q.is_req_new_pack,
		q.new_pack_desc,
		q.createddate,
		q.createdby,
		initcap(p.productname) as coffeetype,
		b.incoterms,
		initcap(d.destination) as destination,
		q.new_pack_task_status,
		q.margin,
		q.marginpercentage,
		q.basepricekgs,
		q.finalprice,
		q.custapprove,
		q.gmsapprovalstatus,
		q.rejectionremarks,
		q.gmc_rej_comments,
		q.confirmedorderquantity_kgs,
		q.negativemarginstatus,
		q.negativemarginremarks,
		q.negativemarginreason
	from dbo.cms_quote_item_master q
	INNER JOIN dbo.crm_quote_master w on q.quoteid = w.quoteid
	LEFT JOIN dbo.cms_incoterms_master b on w.incotermsid = b.incotermsid
	INNER JOIN dbo.qua_product_sample_master h on q.sampleid = h.sampleid
	LEFT JOIN dbo.prod_product_master p on p.productid = h.productid
	LEFT JOIN dbo.cms_destination_master d ON d.destinationid = w.destinationid
	where quoteitemid=$1`
		rows, err = db.Query(sqlStatement, input.QuoteLineItemId)

		if err != nil {
			log.Println(err)
			log.Println(err, "Not able to get information from quote line item table")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		var lineItem QuoteItemDetails
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(
				&lineItem.QuoteId,
				&lineItem.QuoteNumber,
				&lineItem.QuoteLineItemNumber,
				&lineItem.SampleId,
				&lineItem.SampleCode,
				&lineItem.ExpectedOrder,
				&packingId,
				&packCategoryTypeId,
				&packWeightId,
				&packupcId,
				&packCartonId,
				&packCapTypeId,
				&packSecondaryId,
				&packNoofSecondaryId,
				&parllelizationRequired,
				&customerBrandName,
				&additionalRequirement,
				&isReqNewPack,
				&packDescription,
				&lineItem.CreatedDate,
				&lineItem.CreatedBy,
				&productName,
				&lineItem.InCoterms,
				&destination, &newPackTypeRequestStatus, &margin, &marginPercenatge, &basePrice, &finalPrice, &custApprove, &gmsApprovalStatus,
				&rejectionRemarks, &gmsRejectionComments, &confirmedOrderQuantity,
				&nmstatus, &nmremarks, &nmreason)
		}

		lineItem.CustomerBrandName = customerBrandName.String
		lineItem.AdditionalRequirements = additionalRequirement.String
		lineItem.Destination = destination.String
		lineItem.CoffeeType = productName.String
		lineItem.NewPackTypeRequestStatus = newPackTypeRequestStatus.String
		lineItem.Palletizationrequireid, _ = strconv.Atoi(parllelizationRequired.String)
		lineItem.PackingId, _ = strconv.Atoi(packingId.String)
		lineItem.PackingTypeId, _ = strconv.Atoi(packCategoryTypeId.String)
		lineItem.WeightId, _ = strconv.Atoi(packWeightId.String)
		lineItem.SecondaryId, _ = strconv.Atoi(packSecondaryId.String)
		lineItem.NoOfSecondaryId, _ = strconv.Atoi(packNoofSecondaryId.String)
		lineItem.UPCId, _ = strconv.Atoi(packupcId.String)
		lineItem.CartonTypeId, _ = strconv.Atoi(packCartonId.String)
		lineItem.CapTypeId, _ = strconv.Atoi(packCapTypeId.String)
		lineItem.IsReqNewPacking, _ = strconv.Atoi(isReqNewPack.String)
		lineItem.NewPackingDescription = packDescription.String

		lineItem.Margin = margin.String
		lineItem.MarginPercentage = marginPercenatge.String
		lineItem.BasePrice = basePrice.String
		lineItem.FinalPrice = finalPrice.String
		lineItem.CustApprove = custApprove.String
		lineItem.GmsApprovalStatus = gmsApprovalStatus.String
		lineItem.CustomerRejectionRemarks = rejectionRemarks.String
		lineItem.GmsRejectionRemarks = gmsRejectionComments.String
		lineItem.ConfirmedOrderQuantity = confirmedOrderQuantity.String
		lineItem.NegativeMarginStatus = nmstatus.String
		lineItem.NegativeMarginRemarks = nmremarks.String
		lineItem.NegativeMarginReason = nmreason.String

		log.Println("NOof secondary id-", lineItem.NoOfSecondaryId)
		if lineItem.PackingId != 0 {
			sqlStatement := `select categoryname from dbo.cms_prod_pack_category where categoryid=$1`
			rows8, err := db.Query(sqlStatement, lineItem.PackingId)

			if err != nil {
				log.Println(err)
			}

			for rows8.Next() {
				err = rows8.Scan(&lineItem.CategoryName)
			}
		}
		if lineItem.PackingTypeId != 0 {
			sqlStatement := `select categorytypename from dbo.cms_prod_pack_category_type where categorytypeid=$1`
			rows1, err := db.Query(sqlStatement, lineItem.PackingTypeId)

			if err != nil {
				log.Println(err)
			}

			for rows1.Next() {
				err = rows1.Scan(&lineItem.CategoryTypeName)
			}
		}
		if lineItem.WeightId != 0 {
			sqlStatement := `select weightname from dbo.cms_prod_pack_category_weight where weightid=$1`
			rows2, err := db.Query(sqlStatement, lineItem.WeightId)

			if err != nil {
				log.Println(err)
			}

			for rows2.Next() {
				err = rows2.Scan(&lineItem.WeightName)
			}
		}
		if lineItem.SecondaryId != 0 {
			sqlStatement := `select packsecondaryname from dbo.cms_prod_pack_packsecondary where packsecondaryid=$1`
			rows3, err := db.Query(sqlStatement, lineItem.SecondaryId)

			if err != nil {
				log.Println(err)
			}

			for rows3.Next() {
				err = rows3.Scan(&lineItem.SecondaryName)
			}
		}
		if lineItem.NoOfSecondaryId != 0 {
			sqlStatement := `select noofsecondaryname from dbo.cms_prod_pack_noofsecondary where noofsecondaryid=$1`
			rows4, err := db.Query(sqlStatement, lineItem.NoOfSecondaryId)

			if err != nil {
				log.Println(err)
			}

			for rows4.Next() {
				err = rows4.Scan(&lineItem.NoOfSecondaryName)
			}
		}
		if lineItem.UPCId != 0 {
			sqlStatement := `select upcname from dbo.cms_prod_pack_upc where upcid=$1`
			rows5, err := db.Query(sqlStatement, lineItem.UPCId)

			if err != nil {
				log.Println(err)
			}

			for rows5.Next() {
				err = rows5.Scan(&lineItem.UPCName)
			}
		}
		if lineItem.CartonTypeId != 0 {
			sqlStatement := `select cartontypename from dbo.cms_prod_pack_cartontype where cartontypeid=$1`
			rows6, err := db.Query(sqlStatement, lineItem.CartonTypeId)

			if err != nil {
				log.Println(err)
			}

			for rows6.Next() {
				err = rows6.Scan(&lineItem.CartonTypeName)
			}
		}
		if lineItem.CapTypeId != 0 {
			sqlStatement := `select captypename from dbo.cms_prod_pack_captype where captypeid=$1`
			rows7, err := db.Query(sqlStatement, lineItem.CapTypeId)

			if err != nil {
				log.Println(err)
			}

			for rows7.Next() {
				err = rows7.Scan(&lineItem.CapTypeName)
			}
		}

		//---------------------Fetch Audit Log Info-------------------------------------//
		log.Println("Fetching Audit Log Info #")
		sqlStatementAI := `select u.username as createduser, a.created_date,
			a.description,a.status, v.username as modifieduser, a.modified_date
		   from dbo.auditlog_cms_quote_item_master_newpg a
		   inner join dbo.users_master_newpg u on a.createdby=u.userid
		   left join dbo.users_master_newpg v on a.modifiedby=v.userid
		   where quoteitemid=$1 order by logid desc limit 1`
		rowsAI, errAI := db.Query(sqlStatementAI, input.QuoteLineItemId)
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
			auditDetails := append(lineItem.AuditLogDetails, al)
			lineItem.AuditLogDetails = auditDetails
			log.Println("added one")

		}
		log.Println("Audit Details:", lineItem.AuditLogDetails)

		res, _ = json.Marshal(lineItem)
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getQuoteLineCreationInfo)
}
