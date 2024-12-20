package lunamediaV2

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"net/http"
)

type LunaAdapter struct {
	URI string
}

type lunamediaBidExt struct {
	BidType *int `json:"BidType,omitempty"`
}

type respExt struct {
	FledgeAuctionConfigs map[string]json.RawMessage `json:"fledge_auction_configs,omitempty"`
}

func (a *LunaAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errors []error

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errors
}

func (a *LunaAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {

			typedBid := &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			}

			var bidExt *lunamediaBidExt

			//var bidType openrtb_ext.BidType

			//if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
			//	return nil, []error{&errortypes.BadServerResponse{
			//		Message: "Missing BidExt",
			//	}}
			//}
			err := jsonutil.Unmarshal(bid.Ext, &bidExt)
			if err != nil {
				errs = append(errs, err)
			} else if bidExt != nil {
				typedBid.BidType = getBidType(bidExt)
			}
			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
		//if bidResp.Ext != nil {
		//	var bidRespExt respExt
		//	if err := jsonutil.Unmarshal(bidResp.Ext, &bidRespExt); err == nil && bidRespExt.FledgeAuctionConfigs != nil {
		//		bidResponse.FledgeAuctionConfigs = make([]*openrtb_ext.FledgeAuctionConfig, 0, len(bidRespExt.FledgeAuctionConfigs))
		//		for impId, config := range bidRespExt.FledgeAuctionConfigs {
		//			fledgeAuctionConfig := &openrtb_ext.FledgeAuctionConfig{
		//				ImpId:  impId,
		//				Config: config,
		//			}
		//			bidResponse.FledgeAuctionConfigs = append(bidResponse.FledgeAuctionConfigs, fledgeAuctionConfig)
		//		}
		//	}
		//}
	}
	return bidResponse, errs
}

// getBidType returns the bid type specified in the response bid.ext
func getBidType(bidExt *lunamediaBidExt) openrtb_ext.BidType {
	// setting "banner" as the default bid type
	bidType := openrtb_ext.BidTypeBanner
	if bidExt != nil && bidExt.BidType != nil {
		switch *bidExt.BidType {
		case 0:
			bidType = openrtb_ext.BidTypeBanner
		case 1:
			bidType = openrtb_ext.BidTypeVideo
		case 2:
			bidType = openrtb_ext.BidTypeNative
		default:
			bidType = openrtb_ext.BidTypeBanner
		}
	}
	return bidType
}

// Builder builds a new instance of the Pubmatic adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &LunaAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
