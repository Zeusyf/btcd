// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2018-2021 The Omegasuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/zeusyf/btcd/btcec"
	"github.com/zeusyf/btcd/chaincfg"
	"github.com/zeusyf/btcd/txscript/txsparser"
	"github.com/zeusyf/btcd/wire"
	"github.com/zeusyf/btcd/wire/common"
	"github.com/zeusyf/btcutil"
	"github.com/zeusyf/omega/ovm"
)

// RawTxInWitnessSignature returns the serialized ECDA signature for the input
// idx of the given transaction, with the hashType appended to it. This
// function is identical to RawTxInSignature, however the signature generated
// signs a new sighash digest defined in BIP0143.
func RawTxInSignature(tx *wire.MsgTx, idx int, subScript []byte,
	key *btcec.PrivateKey, chainParams *chaincfg.Params) ([]byte, error) {

	hash, err := ovm.CalcSignatureHash(tx, idx, subScript, 0, chainParams)
	if err != nil {
		return nil, err
	}

	signature, err := key.Sign(hash[:])
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return signature.Serialize(), nil
}

// SignatureScript creates an input signature script for tx to spend OMC sent
// from a previous output to the owner of privKey. tx must include all
// transaction inputs and outputs, however txin scripts are allowed to be filled
// or empty. The returned script is calculated to be used as the idx'th txin
// sigscript for tx. subscript is the PkScript of the previous output being used
// as the idx'th input. privKey is serialized in either a compressed or
// uncompressed format based on compress. This format must match the same format
// used to generate the payment address, or the script validation will fail.
func SignatureScript(tx *wire.MsgTx, idx int, subscript []byte, privKey *btcec.PrivateKey,
	compress bool, chainParams *chaincfg.Params, hashType SigHashType) ([]byte, error) {
	script := ovm.NewScriptBuilder()

	// generate header data for preparing signature hash
	var payscript []byte
	switch subscript[21] {
	case ovm.OP_PAY2PKH:
		payscript = []byte{byte(ovm.SIGNTEXT), byte(hashType)}
	case ovm.OP_PAYMULTISIG:
		return []byte{}, fmt.Errorf("Internal error: SignatureScript does not support multi signature yet")
	default:
		return []byte{}, fmt.Errorf("Internal error: call SignatureScript while pkScript does not require signature")
	}

	sig, err := RawTxInSignature(tx, idx, payscript, privKey, chainParams)
	if err != nil {
		return nil, err
	}

	pk := (*btcec.PublicKey)(&privKey.PublicKey)
	var pkData []byte
	if compress {
		pkData = pk.SerializeCompressed()
	} else {
		pkData = pk.SerializeUncompressed()
	}

	script.AddOp(ovm.PUSH, []byte{0}).AddByte(byte(len(pkData))).AddBytes(pkData)
	script.AddOp(ovm.PUSH, []byte{0}).AddByte(byte(len(sig))).AddBytes(sig)
	script.AddOp(ovm.SIGNTEXT, []byte{byte(hashType)})

	return script.Script(), nil
}

func extractMSscript(subScript []byte, chainParams *chaincfg.Params) []byte {
	builder := ovm.NewScriptBuilder()

	builder.AddBytes(subScript[:8]) // PUSH 4 M N SIGNTEXT textcode

	nsigs := int(common.LittleEndian.Uint16(subScript[2:4]))

	for i := 8; i < len(subScript) && nsigs > 0; {
		switch ovm.OpCode(subScript[i]) {
		case ovm.PUSH:
			pushed := subScript[i+1]
			if pushed != 25 || subScript[i+2] != chainParams.MultiSigAddrID {
				for subScript[i] == byte(ovm.PUSH) {
					pushed = subScript[i+1]
					builder.AddOp(ovm.PUSH, []byte{subScript[i+1]}).
						AddBytes(subScript[i+2 : i+2+int(pushed)])
					i += 2 + int(pushed)
				}
				continue
			}

			builder.AddOp(ovm.PUSH, []byte{subScript[i+1]}).
				AddBytes(subScript[i+2 : i+2+int(pushed)])

			i += 2 + int(pushed)

			if subScript[i] == byte(ovm.SIGNTEXT) {
				continue
			}

			script := extractMSscript(subScript[i:], chainParams)
			i += len(script)

			for j := 0; j < len(script); {
				if script[j] == byte(ovm.PUSH) {
					pushed := subScript[j+1]
					builder.AddOp(ovm.PUSH, []byte{subScript[j+1]}).
						AddBytes(subScript[j+2 : j+2+int(pushed)])
					j += 2 + int(pushed)
				} else if script[j] == byte(ovm.SIGNTEXT) {
					builder.AddOp(ovm.SIGNTEXT, []byte{subScript[j+1]})
					j += 2
				}
			}

		case ovm.SIGNTEXT:
			nsigs--
			builder.AddOp(ovm.SIGNTEXT, []byte{subScript[i+1]})
			i += 2
		}
	}

	script := builder.Script()
	return script
}

// signMultiSig signs as many of the outputs in the provided multisig script as
// possible. It returns the generated script and a boolean if the script fulfils
// the contract (i.e. nrequired signatures are provided).  Since it is arguably
// legal to not be able to sign any of the outputs, no error is returned.
func signMultiSig(tx *wire.MsgTx, idx int, subScript []byte,
	kdb KeyDB, sdb ScriptDB,
	chainParams *chaincfg.Params, hashType SigHashType) ([]byte, int, bool) {
	// generate header data for preparing signature hash
	// subScript is either a redeem script, or a previously signed sig script
	var payscript []byte
	builder := ovm.NewScriptBuilder()

	// PUSH 4 M N SIGNTEXT textcode
	builder.AddOp(ovm.PUSH, []byte{subScript[1]}).
		AddBytes(subScript[2 : 2+int(subScript[1])])
	builder.AddOp(ovm.SIGNTEXT, []byte{subScript[7]})

	signed := 0
	nRequired := int(common.LittleEndian.Uint16(subScript[4:6]))
	payscript = []byte{byte(ovm.SIGNTEXT), byte(hashType)}

	for i := 8; i < len(subScript); {
		switch ovm.OpCode(subScript[i]) {
		case ovm.PUSH:
			pushed := subScript[i+1]
			if pushed != 21 ||
				(subScript[i+2] != chainParams.PubKeyHashAddrID &&
					subScript[i+2] != chainParams.MultiSigAddrID &&
					subScript[i+2] != chainParams.ScriptHashAddrID) {
				// it is not lock script. copy data until SIGNTEXT
				for subScript[i] == byte(ovm.PUSH) {
					pushed = subScript[i+1]
					builder.AddOp(ovm.PUSH, []byte{pushed}).
						AddBytes(subScript[i+2 : i+2+int(pushed)])
					i += 2 + int(pushed)
				}
				// it is a signature by others.
				signed++
				continue
			}

			switch subScript[i+2] {
			case chainParams.PubKeyHashAddrID:
				pkscript := subScript[i+3 : i+23]
				kadr, _ := btcutil.NewAddressPubKeyHash(pkscript, chainParams)
				key, _, err := kdb.GetKey(kadr)

				if err != nil || key == nil {
					// no matching key
					builder.AddOp(ovm.PUSH, []byte{pushed}).
						AddBytes(subScript[i+2 : i+2+int(pushed)])
					i = i + 2 + int(pushed)
					continue
				}

				sig, err := RawTxInSignature(tx, idx, payscript, key, chainParams)
				if err != nil || sig == nil {
					// no sig
					builder.AddOp(ovm.PUSH, []byte{pushed}).
						AddBytes(subScript[i+2 : i+2+int(pushed)])
					i = i + 2 + int(pushed)
					continue
				}

				// use signature to replace it

				pk := (*btcec.PublicKey)(&key.PublicKey)

				// choice of format, compressed or not
				pkData := pk.SerializeCompressed()
				hk := btcutil.Hash160(pkData)
				if bytes.Compare(pkscript, hk[:]) != 0 {
					pkData = pk.SerializeUncompressed()
				}

				builder.AddOp(ovm.PUSH, []byte{0}).AddByte(byte(len(pkData))).AddBytes(pkData)
				builder.AddOp(ovm.PUSH, []byte{0}).AddByte(byte(len(sig))).AddBytes(sig)
				builder.AddOp(ovm.SIGNTEXT, []byte{byte(hashType)})
				signed++

				i += 23
				if ovm.OpCode(subScript[i]) == ovm.SIGNTEXT {
					// always
					i += 2
				}
				if signed >= nRequired {
					break
				}

			case chainParams.MultiSigAddrID:
				builder.AddOp(ovm.PUSH, []byte{pushed}).
					AddBytes(subScript[i+2 : i+2+int(pushed)])

				var script []byte

				if subScript[i+23] == byte(ovm.SIGNTEXT) {
					// a new MS script, try to find redeem script
					addr, _ := btcutil.NewAddressMultiSig(subScript[i+3:i+23], chainParams)

					i = 2 + int(pushed)

					if addr == nil {
						continue
					}
					script, _ = sdb.GetScript(addr)
					if script == nil {
						continue
					}
				} else {
					script = extractMSscript(subScript[i+23:], chainParams)
					i = 2 + int(pushed) + len(script)
				}

				sig, _, res := signMultiSig(tx, idx, script, kdb, sdb, chainParams, hashType)
				// merge script
				for j := 0; j < len(sig); {
					if sig[j] == byte(ovm.PUSH) {
						pushed := subScript[j+1]
						builder.AddOp(ovm.PUSH, []byte{subScript[j+1]}).
							AddBytes(subScript[j+2 : j+2+int(pushed)])
						j += 2 + int(pushed)
					} else if sig[j] == byte(ovm.SIGNTEXT) {
						builder.AddOp(ovm.SIGNTEXT, []byte{subScript[j+1]})
						j += 2
					}
				}
				if res {
					signed++
				}
				if signed >= nRequired {
					break
				}

			case chainParams.ScriptHashAddrID:
				if subScript[i+23] == byte(ovm.PUSH) {
					// already signed by others, copy data until next SIGNTEXT
					for subScript[i] == byte(ovm.PUSH) {
						pushed := subScript[i+1]
						builder.AddOp(ovm.PUSH, []byte{pushed}).
							AddBytes(subScript[i+2 : i+2+int(pushed)])
						i += 2 + int(pushed)
					}
					continue
				}

				builder.AddOp(ovm.PUSH, []byte{pushed}).
					AddBytes(subScript[i+2 : i+2+int(pushed)])

				ps, err := btcutil.NewAddressScriptHash(subScript[i+3:i+23], chainParams)
				if err != nil {
					continue
				}

				i += 2 + int(pushed)

				script, err := sdb.GetScript(ps)
				if script == nil || err != nil {
					continue
				}

				k := len(script)
				for k != 0 {
					if k <= 255 {
						builder.AddOp(ovm.PUSH, []byte{byte(k)}).AddBytes(script)
						k = 0
					} else {
						builder.AddOp(ovm.PUSH, []byte{255}).AddBytes(script[:255])
						script = script[255:]
						k = len(script)
					}
				}
				signed++
				if signed >= nRequired {
					break
				}
			}

		case ovm.SIGNTEXT:
			builder.AddOp(ovm.SIGNTEXT, []byte{subScript[i+1]})
			i += 2

		default:
			return nil, 0, false
		}
	}

	if signed == 0 {
		return nil, nRequired, false
	}

	script := builder.Script()
	return script, nRequired, signed >= nRequired
}

func sign(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int,
	subScript []byte, kdb KeyDB, sdb ScriptDB, previousScript []byte, hashType SigHashType) ([]byte, txsparser.ScriptClass, []btcutil.Address, int, error) {

	class, addresses, nrequired, err := ExtractPkScriptAddrs(subScript, chainParams)
	if err != nil {
		return nil, class, nil, 0, err
	}

	switch class {
	case txsparser.PubKeyHashTy:
		// look up key for address
		key, compressed, err := kdb.GetKey(addresses[0])
		if err != nil || key == nil {
			return nil, class, nil, 0, err
		}

		script, err := SignatureScript(tx, idx, subScript, key, compressed, chainParams, hashType)
		if err != nil {
			return nil, class, nil, 0, err
		}

		return script, class, addresses, nrequired, nil

	case txsparser.ScriptHashTy:
		script, err := sdb.GetScript(addresses[0])
		if err != nil || script == nil {
			return nil, class, nil, 0, err
		}

		return script, class, addresses, nrequired, nil

	case txsparser.MultiSigTy:
		if len(previousScript) > 0 {
			script, nrequired, _ := signMultiSig(tx, idx, previousScript,
				kdb, sdb, chainParams, hashType)
			return script, class, addresses, nrequired, nil
		} else {
			script, err := sdb.GetScript(addresses[0])
			if err != nil || script == nil {
				return nil, class, nil, 0, err
			}

			script, nrequired, _ := signMultiSig(tx, idx, script,
				kdb, sdb, chainParams, hashType)
			return script, class, addresses, nrequired, nil
		}

	case txsparser.NullDataTy:
		return nil, class, nil, 0,
			errors.New("can't sign NULLDATA transactions")

	default:
		return nil, class, nil, 0,
			errors.New("can't sign unknown transactions")
	}
}

// KeyDB is an interface type provided to SignTxOutput, it encapsulates
// any user state required to get the private keys for an address.
type KeyDB interface {
	GetKey(btcutil.Address) (*btcec.PrivateKey, bool, error)
}

// KeyClosure implements KeyDB with a closure.
type KeyClosure func(btcutil.Address) (*btcec.PrivateKey, bool, error)

// GetKey implements KeyDB by returning the result of calling the closure.
func (kc KeyClosure) GetKey(address btcutil.Address) (*btcec.PrivateKey,
	bool, error) {
	return kc(address)
}

// ScriptDB is an interface type provided to SignTxOutput, it encapsulates any
// user state required to get the scripts for an pay-to-script-hash address.
type ScriptDB interface {
	GetScript(btcutil.Address) ([]byte, error)
}

// ScriptClosure implements ScriptDB with a closure.
type ScriptClosure func(btcutil.Address) ([]byte, error)

// GetScript implements ScriptDB by returning the result of calling the closure.
func (sc ScriptClosure) GetScript(address btcutil.Address) ([]byte, error) {
	return sc(address)
}

// SignTxOutput signs output idx of the given tx to resolve the script given in
// pkScript with a signature type of hashType. Any keys required will be
// looked up by calling getKey() with the string of the given address.
// Any pay-to-script-hash signatures will be similarly looked up by calling
// getScript. If previousScript is provided then the results in previousScript
// will be merged in a type-dependent manner with the newly generated.
// signature script.
func SignTxOutput(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int,
	pkScript []byte, hashType SigHashType, kdb KeyDB, sdb ScriptDB,
	previousScript []byte) ([]byte, error) {

	sigScript, class, _, _, err := sign(chainParams, tx,
		idx, pkScript, kdb, sdb, previousScript, hashType)
	if err != nil {
		return nil, err
	}

	if class == txsparser.ScriptHashTy {
		// TODO keep the sub addressed and pass down to merge.
		realSigScript, _, _, _, err := sign(chainParams, tx, idx,
			sigScript, kdb, sdb, previousScript, hashType)
		if err != nil {
			return nil, err
		}

		if realSigScript != nil {
			// Append the p2sh script as the last push in the script.
			builder := ovm.NewScriptBuilder().AddOp(ovm.PUSH, []byte{0}).AddBytes(realSigScript)

			sigScript = builder.Script()
		}
		// TODO keep a copy of the script for merging.
	}

	return sigScript, nil

	// Merge scripts. with any previous data, if any.
	//	mergedScript := mergeScripts(chainParams, tx, idx, pkScript, class,	addresses, nrequired, sigScript, previousScript)
	//	mergedScript := mergeScripts(class, sigScript, previousScript)
	//	return mergedScript, nil
}

// mergeScripts merges sigScript and prevScript assuming they are both
// partial solutions for pkScript spending output idx of tx. class, addresses
// and nrequired are the result of extracting the addresses from pkscript.
// The return value is the best effort merging of the two scripts. Calling this
// function with addresses, class and nrequired that do not match pkScript is
// an error and results in undefined behaviour.
/*
func mergeScripts(class txsparser.ScriptClass, sigScript, prevScript []byte) []byte {
	if len(prevScript) == 0 {
		return sigScript
	}
	if len(sigScript) == 0 {
		return prevScript
	}

	switch class {
	case txsparser.MultiSigTy:
		return append(prevScript, sigScript...)
/*
		p, h, err := ExtractSigHead(prevScript)
		if err != nil {
			return prevScript
		}
		p2, h2, err := ExtractSigHead(sigScript)
		if err != nil {
			return prevScript
		}
		if bytes.Compare(h, h2) != 0 {
			return prevScript
		}

		return append(append(h, prevScript[p:]...), sigScript[p2:]...)
 * /

	default:
		if len(sigScript) > len(prevScript) {
			return sigScript
		}
		return prevScript
	}
}
*/
