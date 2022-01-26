package integrations

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"

	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"

	"github.com/stellar/starbridge/cmd/starbridge/model"
)

// TODO NS need to set the contract account, source account from a config file
var sourceAccount = "GAEGI7MPXUMSFS2CHBU46LV6SFHHHYNONW5OM3MTGCRVIQCSSXOB5KOW" // var sourceSecretKey = "SASII4SLKZ3S2GD52BILLO5BC7P45C3RYKOH5BADLSIJWHCUTIAQHYSZ"

// GetSourceAccount() fetches the source account
func GetSourceAccount() string {
	return sourceAccount
}

var baseFee int64 = 100

func getStellarAsset(assetInfo *model.AssetInfo) txnbuild.Asset {
	if assetInfo.ContractAddress == "native" {
		return txnbuild.NativeAsset{}
	}
	return txnbuild.CreditAsset{
		Code:   assetInfo.Code,
		Issuer: assetInfo.ContractAddress,
	}
}

func Transaction2Stellar(tx *model.Transaction) (*txnbuild.Transaction, error) {
	if tx.Chain != model.ChainStellar {
		return nil, fmt.Errorf("cannot convert transaction from a different chain ('%s') to Stellar, need to convert the model.Transaction to the Stellar chain first", tx.Chain.Name)
	}

	if tx.Data.TargetDestinationChain != model.ChainStellar {
		return nil, fmt.Errorf("stellar needs to be the destination chain (found=%s), we should not be dealing with native Stellar transactions in the codebase until we want to submit", tx.Data.TargetDestinationChain)
	}

	ops := []txnbuild.Operation{}
	decimalStringFormat := fmt.Sprintf("%%.%df", tx.AssetInfo.Decimals)
	amountAsDecimalString := fmt.Sprintf(decimalStringFormat, float64(tx.Amount)/math.Pow10(tx.AssetInfo.Decimals))
	ops = append(ops, &txnbuild.CreateClaimableBalance{
		Destinations: []txnbuild.Claimant{
			txnbuild.NewClaimant(tx.To, &txnbuild.UnconditionalPredicate),
		},
		Asset:         getStellarAsset(tx.AssetInfo),
		Amount:        amountAsDecimalString,
		SourceAccount: tx.From, // specify the account here since we use a different source account on the Stellar tx
	})

	return txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: sourceAccount,
				Sequence:  int64(tx.SeqNum),
			},
			BaseFee:              baseFee,
			IncrementSequenceNum: false,
			Operations:           ops,
			Timebounds:           txnbuild.NewInfiniteTimeout(),
		},
	)
}

// Stellar2String is a String converter
func Stellar2String(tx *txnbuild.Transaction) string {
	memoString := ""
	if tx.Memo() != nil {
		memoXdr, _ := tx.Memo().ToXDR()
		memoString = memoXdr.GoString()
	}

	sb := strings.Builder{}
	sb.WriteString("StellarTx[")
	sb.WriteString(fmt.Sprintf("SourceAccount=%s", tx.SourceAccount().AccountID))
	sb.WriteString(fmt.Sprintf(", SeqNum=%d", tx.SequenceNumber()))
	sb.WriteString(fmt.Sprintf(", BaseFee=%d", tx.BaseFee()))
	sb.WriteString(fmt.Sprintf(", MaxFee=%d", tx.MaxFee()))
	sb.WriteString(fmt.Sprintf(", TimeBounds.MinTime=%d", tx.Timebounds().MinTime))
	sb.WriteString(fmt.Sprintf(", TimeBounds.MaxTime=%d", tx.Timebounds().MaxTime))
	sb.WriteString(fmt.Sprintf(", Memo=%s", memoString))
	sb.WriteString(fmt.Sprintf(", Operations=%s", stellarOps2String(tx.Operations())))
	sb.WriteString(fmt.Sprintf(", Signatures=%s", stellarSigs(tx.Signatures())))
	sb.WriteString("]")
	return sb.String()
}

func stellarOps2String(ops []txnbuild.Operation) string {
	sb := strings.Builder{}
	sb.WriteString("Array[")

	for i := 0; i < len(ops); i++ {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(stellarOp2String(ops[i]))
	}

	sb.WriteString("]")
	return sb.String()
}

func stellarSigs(sigs []xdr.DecoratedSignature) string {
	sb := strings.Builder{}
	sb.WriteString("Array[")

	for i := 0; i < len(sigs); i++ {
		if i != 0 {
			sb.WriteString(", ")
		}
		sigOut := base64.StdEncoding.EncodeToString(sigs[i].Signature)
		sb.WriteString(sigOut)
	}

	sb.WriteString("]")
	return sb.String()
}

func stellarOp2String(op txnbuild.Operation) string {
	sb := strings.Builder{}
	switch o := op.(type) {
	case *txnbuild.CreateClaimableBalance:
		sb.WriteString("Operation[")
		sb.WriteString(fmt.Sprintf("SourceAccount=%s", op.GetSourceAccount()))
		sb.WriteString(fmt.Sprintf(", Type=%s", "CreateClaimableBalance"))
		sb.WriteString(fmt.Sprintf(", Destinations=%s", getDestinationsString(o.Destinations)))
		sb.WriteString(fmt.Sprintf(", Amount=%s", o.Amount))

		asset := ""
		if o.Asset.IsNative() {
			asset = "native"
		} else {
			asset = fmt.Sprintf("%s:%s", o.Asset.GetCode(), o.Asset.GetIssuer())
		}
		sb.WriteString(fmt.Sprintf(", Asset=%s", asset))
		sb.WriteString("]")
	default:
		sb.WriteString(fmt.Sprintf("unrecognized_operation_type__%T", o))
	}
	return sb.String()
}

func getDestinationsString(destinations []txnbuild.Claimant) string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, d := range destinations {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("Claimant[")
		sb.WriteString(fmt.Sprintf("Desination=%s", d.Destination))
		sb.WriteString(fmt.Sprintf(", Predicate=%s", d.Predicate.Type.String()))
		sb.WriteString("]")
	}
	sb.WriteString("]")
	return sb.String()
}
