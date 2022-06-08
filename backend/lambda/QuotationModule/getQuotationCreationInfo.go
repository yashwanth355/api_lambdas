package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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

type QuoteDetails struct {
	QuoteGeneratedNumber   string     `json:"quote_autogennumber"`
	Accountid              string     `json:"accountid"`
	Accountname            string     `json:"accountname"`
	Accounttypename        string     `json:"accounttypename"`
	Contactname            string     `json:"contactname"`
	Contactid              string     `json:"contactid"`
	Currencycode           string     `json:"currencycode"`
	Currencyid             string     `json:"currencyid"`
	Paymentterms           string     `json:"payment_terms"`
	Remarksfrommarketing   string     `json:"remarks_marketing"`
	Remarksfromgmc         string     `json:"remarks_gmc"`
	Destination            string     `json:"destination_port"`
	PortLoading            string     `json:"port_loading"`
	Otherspecifications    string     `json:"other_specification"`
	Billingaddress         string     `json:"billing_address"`
	Destinationcountryid   string     `json:"destination_countryid"`
	Destinationcountry     string     `json:"destination_country"`
	CreatedDate            string     `json:"createddate"`
	Createdby              string     `json:"createdby"`
	Incoterms              string     `json:"incoterms"`
	Incotermsid            string     `json:"incotermsid"`
	Finalclientaccountid   string     `json:"finalclientaccountid"`
	Finalclientaccountname string     `json:"finalclientaccountname"`
	Portloadingid          string     `json:"portloadingid"`
	Portdestinationid      string     `json:"destinationid"`
	IsPriceRequested       int        `json:"ispricerequested"`
	Currencyname           string     `json:"currencyname"`
	Fromdate               string     `json:"fromdate"`
	Todate                 string     `json:"todate"`
	Status                 string     `json:"status"`
	BillingId              string     `json:"billing_id"`
	PendingWithUser        string     `json:"pending_withuser"`
	PendingWithUserId      string     `json:"pending_withuserid"`
	AuditLogDetails        []AuditLog `json:"audit_log"`
}

type AuditLog struct {
	CreatedDate      string `json:"created_date"`
	CreatedUserName  string `json:"created_username"`
	ModifiedDate     string `json:"modified_date"`
	ModifiedUserName string `json:"modified_username"`
	Description      string `json:"description"`
	Status           string `json:"status"`
}

type InCotermsInfo struct {
	Incotermsid string `json:"incotermsid"`
	Incoterms   string `json:"incoterms"`
}

type Currencies struct {
	Currencyid   string `json:"currencyid"`
	Currencyname string `json:"currencyname"`
	Currencycode string `json:"currencycode"`
}

type Loadingports struct {
	Id              int    `json:"id"`
	Portlaodingname string `json:"portloading_name"`
}

type Destinationports struct {
	Destinationid int    `json:"id"`
	Destination   string `json:"destination_port"`
}

type AccountDetailsForSampleRequest struct {
	Accountid   string           `json:"account_id"`
	Accountname string           `json:"account_name"`
	Contacts    []ContactDetails `json:"contact_details"`
}

type AccountDetailsForQuotation struct {
	Accountid       string           `json:"account_id"`
	Accounttypeid   string           `json:"accounttype_id"`
	Accounttypename string           `json:"accounttype_name"`
	Accountname     string           `json:"account_name"`
	Contacts        []ContactDetails `json:"contact_details"`
}

type ShippingAddressForSampleRequest struct {
	Accountid       string            `json:"account_id"`
	Contactid       string            `json:"contact_id"`
	ShippingAddress []ShippingAddress `json:"shipping_address"`
}

type ShippingAddress struct {
	ShippingId     string `json:"shipping_id"`
	Address        string `json:"address"`
	PrimaryAddress bool   `json:"primary_address"`
}

type BillingAddressForQuotation struct {
	Accountid      string           `json:"account_id"`
	Contactid      string           `json:"contact_id"`
	BillingAddress []BillingAddress `json:"billing_address"`
}

type BillingAddress struct {
	BillingId      string `json:"billing_id"`
	Address        string `json:"address"`
	PrimaryAddress bool   `json:"primary_address"`
}

type ContactDetails struct {
	Contactid   string `json:"contact_id"`
	Contactname string `json:"contact_name"`
}

type QuotationStatusList struct {
	Id     int    `json:"id"`
	Status string `json:"status"`
}

type Input struct {
	Type          string `json:"type"`
	QuoteId       string `json:"quote_number"`
	Accountid     string `json:"account_id"`
	Contactid     string `json:"contact_id"`
	CreatedUserId string `json:"createdbyuserid"`
}

func getQuotationCreationInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	if input.Type == "incoterms" {
		log.Println("get incoterms", input.Type)
		sqlStatement := `SELECT incotermsid, incoterms FROM dbo.cms_incoterms_master`
		rows, err = db.Query(sqlStatement)

		var allInCoterms []InCotermsInfo
		defer rows.Close()
		for rows.Next() {
			var incoterm InCotermsInfo
			err = rows.Scan(&incoterm.Incotermsid, &incoterm.Incoterms)
			allInCoterms = append(allInCoterms, incoterm)
		}

		res, _ = json.Marshal(allInCoterms)

	} else if input.Type == "currencies" {
		log.Println("get currencies", input.Type)
		sqlStatement := `SELECT currencyid, initcap(currencyname) as name, currencycode FROM dbo.project_currency_master`
		rows, err = db.Query(sqlStatement)

		var allCurrencies []Currencies
		defer rows.Close()
		for rows.Next() {
			var currency Currencies
			err = rows.Scan(&currency.Currencyid, &currency.Currencyname, &currency.Currencycode)
			allCurrencies = append(allCurrencies, currency)
		}

		res, _ = json.Marshal(allCurrencies)
	} else if input.Type == "loadingports" {
		log.Println("get loading ports", input.Type)
		sqlStatement := `SELECT id, initcap(portloadingname) as name FROM dbo.cms_portloading_master`
		rows, err = db.Query(sqlStatement)

		var allPorts []Loadingports
		defer rows.Close()
		for rows.Next() {
			var port Loadingports
			err = rows.Scan(&port.Id, &port.Portlaodingname)
			allPorts = append(allPorts, port)
		}

		res, _ = json.Marshal(allPorts)
	} else if input.Type == "destinationports" {
		log.Println("get destination ports", input.Type)
		sqlStatement := `SELECT destinationid, initcap(destination) as dest FROM dbo.cms_destination_master`
		rows, err = db.Query(sqlStatement)

		var allDestinationPorts []Destinationports
		defer rows.Close()
		for rows.Next() {
			var destination Destinationports
			err = rows.Scan(&destination.Destinationid, &destination.Destination)
			allDestinationPorts = append(allDestinationPorts, destination)
		}

		res, _ = json.Marshal(allDestinationPorts)
	} else if input.Type == "listquotationstatus" {
		log.Println("get quotation status", input.Type)
		sqlStatement := `select id, status from dbo.cms_allstatus_master`
		rows, err = db.Query(sqlStatement)

		var allQuotationStatus []QuotationStatusList
		defer rows.Close()
		for rows.Next() {
			var status QuotationStatusList
			err = rows.Scan(&status.Id, &status.Status)
			allQuotationStatus = append(allQuotationStatus, status)
		}

		res, _ = json.Marshal(allQuotationStatus)
	} else if input.Type == "billingaddressoncontacts" {
		log.Println("get billing address as per accountid and contactid", input.Type)
		sqlStatement := `SELECT billingid, concat(street,' ', city,' ', stateprovince, ' ', postalcode) as address,
		primary_address, accountid, contactid
		FROM dbo.accounts_billing_address_master where contactid=$1 or accountid=$2
				ORDER BY idsno desc`
		rows, err = db.Query(sqlStatement, input.Contactid, input.Accountid)

		var nullstringAddress, primaryAddress, conid sql.NullString

		var allBillingAddress []BillingAddressForQuotation
		defer rows.Close()
		for rows.Next() {
			var address BillingAddressForQuotation
			var billingAddress BillingAddress
			err = rows.Scan(&billingAddress.BillingId, &nullstringAddress, &primaryAddress, &address.Accountid, &conid)
			billingAddress.Address = nullstringAddress.String
			a, _ := strconv.ParseBool(primaryAddress.String)
			billingAddress.PrimaryAddress = a
			address.Contactid = conid.String
			address.BillingAddress = append(address.BillingAddress, billingAddress)
			allBillingAddress = append(allBillingAddress, address)
		}

		res, _ = json.Marshal(allBillingAddress)
	} else if input.Type == "shippingaddressoncontacts" {
		log.Println("get shipping address as per accountid and contactid", input.Type)
		sqlStatement := `SELECT shippingid, concat(street,' ', city,' ', stateprovince, ' ', postalcode) as address,
		primary_address, accountid, contactid
		FROM dbo.accounts_shipping_address_master where contactid=$1 and accountid=$2
				ORDER BY idsno desc`
		rows, err = db.Query(sqlStatement, input.Contactid, input.Accountid)

		var nullstringAddress, primaryAddress, conid sql.NullString

		var allShippingAddress []ShippingAddressForSampleRequest
		defer rows.Close()
		for rows.Next() {
			var address ShippingAddressForSampleRequest
			var shippingAddress ShippingAddress
			err = rows.Scan(&shippingAddress.ShippingId, &nullstringAddress, &primaryAddress, &address.Accountid, &conid)
			shippingAddress.Address = nullstringAddress.String
			a, _ := strconv.ParseBool(primaryAddress.String)
			shippingAddress.PrimaryAddress = a
			address.Contactid = conid.String
			address.ShippingAddress = append(address.ShippingAddress, shippingAddress)
			allShippingAddress = append(allShippingAddress, address)
		}

		res, _ = json.Marshal(allShippingAddress)
	} else if input.Type == "accountdetailsforsamplerequest" {
		log.Println("get account details", input.Type)
		var allAccounts []AccountDetailsForSampleRequest
		var param string
		if input.CreatedUserId == "" {
			param = " order by a.accountid asc"
		} else {
			param = " where a.createduserid=" + "'" + input.CreatedUserId + "'" + " order by a.accountid asc"
		}

		log.Printf("Filter Query:", param)

		sqlStatement := `select distinct a.accountid, initcap(a.accountname) as name
		from dbo.accounts_master a %s`
		rows, err = db.Query(fmt.Sprintf(sqlStatement, param))
		defer rows.Close()
		for rows.Next() {
			var account AccountDetailsForSampleRequest
			err = rows.Scan(&account.Accountid, &account.Accountname)
			if account.Accountid != "" {
				sqlStatement := `SELECT contactid, initcap(contactfirstname) as name FROM dbo.contacts_master where accountid=$1`
				rows2, err2 := db.Query(sqlStatement, account.Accountid)

				if err2 != nil {
					log.Println(err)
					log.Println("unable to add account type", account.Accountid)
				}

				for rows2.Next() {
					var contact ContactDetails
					err = rows2.Scan(&contact.Contactid, &contact.Contactname)
					allAccounts := append(account.Contacts, contact)
					account.Contacts = allAccounts
					log.Println("added one account", allAccounts)
				}
			}
			allAccounts = append(allAccounts, account)

		}

		res, _ = json.Marshal(allAccounts)
	} else if input.Type == "accountdetailsforQuotation" {
		log.Println("get account details", input.Type)
		var allAccounts []AccountDetailsForQuotation
		var param string
		if input.CreatedUserId == "" {
			param = " order by a.accountid asc"
		} else {
			param = " where a.createduserid=" + "'" + input.CreatedUserId + "'" + " order by a.accountid asc"
		}

		log.Printf("Filter Query:", param)
		sqlStatement := `select distinct a.accountid, initcap(a.accountname) as name, a.accounttypeid
		from 
	   dbo.accounts_master a %s`
		rows, err = db.Query(fmt.Sprintf(sqlStatement, param))
		defer rows.Close()
		for rows.Next() {
			var account AccountDetailsForQuotation
			err = rows.Scan(&account.Accountid, &account.Accountname, &account.Accounttypeid)

			var accounttypes []string
			if account.Accounttypeid != "" {
				z := strings.Split(account.Accounttypeid, ",")
				for i, z := range z {
					log.Println("get account name", i, z)
					sqlStatement := `SELECT accounttype FROM dbo.cms_account_type_master where accounttypeid=$1`
					rows1, err1 := db.Query(sqlStatement, z)

					if err1 != nil {
						log.Println(err, "unable to add account names")
					}

					for rows1.Next() {
						var accounttype string
						err = rows1.Scan(&accounttype)
						accounttypes = append(accounttypes, accounttype)
					}
				}
			}
			account.Accounttypename = strings.Join(accounttypes, ",")
			if account.Accountid != "" {
				sqlStatement := `SELECT contactid, initcap(contactfirstname) as name FROM dbo.contacts_master where accountid=$1`
				rows2, err2 := db.Query(sqlStatement, account.Accountid)

				if err2 != nil {
					log.Println(err)
					log.Println("unable to add account type", account.Accountid)
				}

				for rows2.Next() {
					var contact ContactDetails
					err = rows2.Scan(&contact.Contactid, &contact.Contactname)
					allAccounts := append(account.Contacts, contact)
					account.Contacts = allAccounts
					log.Println("added one account", allAccounts)
				}
			}
			allAccounts = append(allAccounts, account)
		}

		res, _ = json.Marshal(allAccounts)
	} else if input.Type == "viewquote" {
		log.Println("get account details", input.Type)
		var contactId, contactName, accountTypeName, createdBy, paymentTerm, otherSpecification,
			remarks, destionationCountryId, destionationCountry, destination, remarksFromGMC, billingId,
			assignedToUserId, assignedToUser, isPriceRequested sql.NullString
		sqlStatement := `SELECT q.quotenumber, q.accountid,a.accountname, q.accounttypename, q.contactid, t.contactfirstname, q.createddate, u.username as createdby , 
        r.currencyname,
        q.currencycode,
        r.currencyid,
        q.fromdate,
		q.todate,
		q.paymentterm,
		q.otherspecification,
		q.remarks,
		q.destinationcountryid,
		m.countryname,
		initcap(d.destination) as destination,
		q.finalaccountid,
		concat(b.street,' ', b.city,' ', b.stateprovince, ' ', b.postalcode) as address,
		c.incoterms,
		q.incotermsid,
		s.status,
		initcap(p.portloadingname) as portloadingname,
        q.portloadingid,
		q.destinationid,
		q.remarksfromgmc, q.billing_id, q.assignedto, v.username, q.ispricerequested from dbo.crm_quote_master q
        INNER JOIN dbo.accounts_master a on q.accountid = a.accountid 
		INNER JOIN dbo.cms_incoterms_master c on q.incotermsid = c.incotermsid
        INNER JOIN dbo.cms_allstatus_master s ON q.statusid = s.id
		LEFT JOIN dbo.users_master_newpg u ON q.createdby = u.userid
        LEFT JOIN dbo.contacts_master t ON q.contactid = t.contactid
		INNER JOIN dbo.project_currency_master r ON q.currencycode = r.currencycode
		INNER JOIN dbo.cms_portloading_master p ON p.id = q.portloadingid
        LEFT JOIN dbo.cms_destination_master d ON d.destinationid = q.destinationid
		LEFT JOIN dbo.accounts_billing_address_master b on q.billing_id = b.billingid
		LEFT JOIN dbo.users_master_newpg v ON q.assignedto = v.userid
		LEFT JOIN dbo.countries_master m ON q.destinationcountryid = cast(m.countryid as varchar)
		where q.quoteid =$1`
		rows, err = db.Query(sqlStatement, input.QuoteId)

		var quote QuoteDetails
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&quote.QuoteGeneratedNumber, &quote.Accountid,
				&quote.Accountname,
				&accountTypeName,
				&contactId,
				&contactName,
				&quote.CreatedDate,
				&createdBy,
				&quote.Currencyname,
				&quote.Currencycode,
				&quote.Currencyid,
				&quote.Fromdate,
				&quote.Todate,
				&paymentTerm,
				&otherSpecification,
				&remarks,
				&destionationCountryId,
				&destionationCountry,
				&destination,
				&quote.Finalclientaccountid,
				&quote.Billingaddress,
				&quote.Incoterms,
				&quote.Incotermsid,
				&quote.Status,
				&quote.PortLoading,
				&quote.Portloadingid,
				&quote.Portdestinationid,
				&remarksFromGMC, &billingId, &assignedToUserId, &assignedToUser, &isPriceRequested)
		}

		quote.Accounttypename = accountTypeName.String
		quote.Createdby = createdBy.String
		quote.Paymentterms = paymentTerm.String
		quote.Remarksfrommarketing = remarks.String
		quote.Destinationcountryid = destionationCountryId.String
		quote.Destination = destination.String
		quote.Remarksfromgmc = remarksFromGMC.String
		quote.Contactid = contactId.String
		quote.Contactname = contactName.String
		quote.Otherspecifications = otherSpecification.String
		quote.BillingId = billingId.String
		quote.PendingWithUserId = assignedToUserId.String
		quote.PendingWithUser = assignedToUser.String
		quote.Destinationcountry = destionationCountry.String
		quote.IsPriceRequested, _ = strconv.Atoi(isPriceRequested.String)

		if quote.Finalclientaccountid != "" {

			sqlStatement := `select accountname as address from dbo.accounts_master b where accountid=$1`

			rows, err = db.Query(sqlStatement, quote.Finalclientaccountid)

			if err != nil {
				log.Println(err, "unable to add final account name", quote.Finalclientaccountid)
			}

			defer rows.Close()
			for rows.Next() {
				err = rows.Scan(&quote.Finalclientaccountname)
				log.Println("added final account name")
			}
		}

		//---------------------Fetch Audit Log Info-------------------------------------//
		log.Println("Fetching Audit Log Info #")
		sqlStatementAI := `select u.username as createduser, a.created_date,
	a.description,a.status, v.username as modifieduser, a.modified_date
   from dbo.auditlog_crm_quote_master_newpg a
   left join dbo.users_master_newpg u on a.createdby=u.userid
   left join dbo.users_master_newpg v on a.modifiedby=v.userid
   where quoteid=$1 order by logid desc limit 1`
		rowsAI, errAI := db.Query(sqlStatementAI, input.QuoteId)
		log.Println("Audit Info Fetch Query Executed")
		if errAI != nil {
			log.Println("Audit Info Fetch Query failed")
			log.Println(errAI.Error())
		}

		var createdUser, createdDate, modifiedBy, modifiedDate sql.NullString

		for rowsAI.Next() {
			var al AuditLog
			errAI = rowsAI.Scan(&createdUser, &createdDate, &al.Description, &al.Status, &modifiedBy, &modifiedDate)
			al.CreatedUserName = createdUser.String
			al.CreatedDate = createdDate.String
			al.ModifiedUserName = modifiedBy.String
			al.ModifiedDate = modifiedDate.String
			auditDetails := append(quote.AuditLogDetails, al)
			quote.AuditLogDetails = auditDetails
			log.Println("added one")

		}
		log.Println("Audit Details:", quote.AuditLogDetails)

		res, _ = json.Marshal(quote)
	}
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getQuotationCreationInfo)
}
