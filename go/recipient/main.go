package recipient

import (
	ecpdksap_v0 "ecpdksap-go/versions/v0"
	ecpdksap_v2 "ecpdksap-go/versions/v2"

	"ecpdksap-go/utils"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	BN254 "github.com/consensys/gnark-crypto/ecc/bn254"
	BN254_fr "github.com/consensys/gnark-crypto/ecc/bn254/fr"
	SECP256K1 "github.com/consensys/gnark-crypto/ecc/secp256k1"
	SECP256K1_fr "github.com/consensys/gnark-crypto/ecc/secp256k1/fr"
)

func Scan(jsonInputString string) (rP []string, rAddr []string, privKeys []string) {

	// ------------------------ Unpacking json

	var recipientInputData RecipientInputData
	json.Unmarshal([]byte(jsonInputString), &recipientInputData)

	Rs_string := recipientInputData.Rs

	var Rs []BN254.G1Affine

	nTotalRuns := 0
	var duration time.Duration

	var viewTagFcn func (*BN254.G1Affine, uint) (string)
	nBytesInViewTag := uint(0)
	if recipientInputData.ViewTagVersion == "none" {
		//note: default values
	} else if recipientInputData.ViewTagVersion == "v0-1byte" {
		viewTagFcn = utils.BN254_G1PointToViewTag
		nBytesInViewTag = 1

	} else if recipientInputData.ViewTagVersion == "v0-2bytes" {
		viewTagFcn = utils.BN254_G1PointToViewTag
		nBytesInViewTag = 2

	} else if recipientInputData.ViewTagVersion == "v1-1byte" {
		viewTagFcn = utils.BN254_G1PointXCoordToViewTag
		nBytesInViewTag = 1
	}

	var viewTagCalcAggregateDuration time.Duration
	var remainingCalcAggregateDuration time.Duration

	for i := 0; i < len(Rs_string); i++ {
		
		RsiX, RsiY := utils.UnpackXY(Rs_string[i])

		var Rsi BN254.G1Affine
		Rsi.X.SetString(RsiX)
		Rsi.Y.SetString(RsiY)

		Rs = append(Rs, Rsi)
	}

	if recipientInputData.Version == "v0" {

		var k, v BN254_fr.Element
		kBytes, _ := hex.DecodeString(recipientInputData.PK_k)
		k.Unmarshal(kBytes)
		vBytes, _ := hex.DecodeString(recipientInputData.PK_v)
		v.Unmarshal(vBytes)
	
		K, _ := utils.BN254_CalcG2PubKey(k)

		startTime := time.Now()

		for i, Rsi := range Rs { 
			

			vTagCalcStart := time.Now()
			vR := utils.BN254_MulG1PointandElement(&Rsi, &v)
			viewTagsUsedAndDontMatch := viewTagFcn != nil && viewTagFcn(&vR, nBytesInViewTag) != recipientInputData.ViewTags[i]
			viewTagCalcAggregateDuration += time.Since(vTagCalcStart)
			
			if viewTagsUsedAndDontMatch { continue }

			nTotalRuns += 1
			
			rCalcStart := time.Now()

			P, _ := ecpdksap_v0.RecipientComputesStealthPubKey(&K, &Rsi, &v);
			rP = append(rP, hex.EncodeToString(P.Marshal()))

			remainingCalcAggregateDuration += time.Since(rCalcStart)
		} 

		duration = time.Since(startTime)

	} else if recipientInputData.Version == "v2" {

		var k SECP256K1_fr.Element
		kBytes, _ := hex.DecodeString(recipientInputData.PK_k)
		k.Unmarshal(kBytes)
		var k_asBigInt  big.Int
		k.BigInt(&k_asBigInt)

		var v BN254_fr.Element
		vBytes, _ := hex.DecodeString(recipientInputData.PK_v)
		v.Unmarshal(vBytes)
		var v_asBigInt  big.Int
		v.BigInt(&v_asBigInt)

		var K SECP256K1.G1Affine
		K.ScalarMultiplicationBase(&k_asBigInt)

		var V BN254.G1Affine
		V.ScalarMultiplicationBase(&v_asBigInt)

		_, _, _, G2_BN254 := BN254.Generators()

		var vR BN254.G1Affine
		var b SECP256K1_fr.Element
		var P SECP256K1.G1Affine
		var kb SECP256K1_fr.Element

		startTime := time.Now()

		for i, Rsi := range Rs { 

			vTagCalcStart := time.Now()
			vR.ScalarMultiplication(&Rsi, &v_asBigInt)
			viewTagsUsedAndDontMatch := viewTagFcn != nil && viewTagFcn(&vR, nBytesInViewTag) != recipientInputData.ViewTags[i]
			viewTagCalcAggregateDuration += time.Since(vTagCalcStart)
			
			if viewTagsUsedAndDontMatch { continue }

			nTotalRuns += 1

			rCalcStart := time.Now()

			S, _ := BN254.Pair([]BN254.G1Affine{vR}, []BN254.G2Affine{G2_BN254})
			b = ecpdksap_v2.Compute_b_asElement(&S)
			
			kb.Mul(&k, &b)

			P = utils.SECP256k1_MulG1PointandElement(&K, &b)

			rP = append(rP, hex.EncodeToString(S.Marshal()))

			rAddr = append(rAddr, ecpdksap_v2.ComputeEthAddress(&P))
			privKeys = append(privKeys, "0x" + kb.Text(16))

			remainingCalcAggregateDuration += time.Since(rCalcStart)
		}

		duration = time.Since(startTime)
	}

	fmt.Println("ECPDKSAP ::: version:", recipientInputData.Version, "ViewTagVersion:", recipientInputData.ViewTagVersion, "; time:", duration)

	sampleSize := time.Duration(len(Rs_string))

	if nTotalRuns != 0 {
		fmt.Println("----> nTotalRuns: ", nTotalRuns, "avgDuration:", duration / sampleSize)
		fmt.Println("Phase 0 avg. duration: ", viewTagCalcAggregateDuration / sampleSize)
		fmt.Println("Phase 1 avg. duration: ", remainingCalcAggregateDuration / time.Duration(nTotalRuns))
	} else {
		fmt.Println("----> nTotalRuns: ", nTotalRuns)
		fmt.Println("Phase 0 avg. duration: ", viewTagCalcAggregateDuration / sampleSize)
	}
	
	return
}
	
type RecipientInputData struct {
	PK_k string `json:"k"`
	PK_v string `json:"v"`
	Rs []string `json:"Rs"`
	Version string 
	ViewTags [] string
	ViewTagVersion string
}