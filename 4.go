package main

import(
	"fmt"
	kec "github.com/ethereum/go-ethereum/crypto"
	"os"
	"io"
	"log"
	"encoding/json"
	"reflect"
	"sync"
	//"runtime"
	"time"
  	//"context"
	"crypto/rand"
	"crypto/ecdsa"
 	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/core"
	"strconv"
  "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	mathlib "math"
	import_contracts "./contracts"
    // import_contracts "./contracts/contracts_p_IC"
)

const waitingTime=40  //max time to complete a stage
const maxLinesToGate=2
const keySize=32
const hashFunctionOutputBitSize=32
const gateTupleSize=32

var totalNumberOfGates int
var noOfInputGates int
var buffer_size int
var noOfFileChunks int


var contractAddress common.Address
var collectorAddress common.Address

type OwnerToCollectorStruct struct {
		EncryptedOutputOfGates [][]byte
		OwnerAddress common.Address
}

type OwnerToContractStruct struct{

	MerkleRootOfFile [hashFunctionOutputBitSize]byte
	MerkleRootOfEncInp [hashFunctionOutputBitSize]byte
	Keycommit [hashFunctionOutputBitSize]byte
	Price big.Int

}


func main(){

	alloc := make(core.GenesisAlloc)

	noOfInputGates=1
	totalNumberOfGates=noOfInputGates*2
	noOfFileChunks=noOfInputGates
	buffer_size=64;



	authOwner,_:=	AuthAndAddressGeneration()
	authCollector,_:=AuthAndAddressGeneration()
	authBuyer,_:=AuthAndAddressGeneration()
	alloc[authOwner.From] = core.GenesisAccount{Balance: big.NewInt(1100000000000000)}
	alloc[authCollector.From] = core.GenesisAccount{Balance: big.NewInt(11000000000)}
	alloc[authBuyer.From] = core.GenesisAccount{Balance: big.NewInt(11000000000)}

	collectorAddress=authCollector.From

	client := backends.NewSimulatedBackend(alloc,60000000000) //client is like blockchain.

	var err error
	contractAddress,_,_,err =import_contracts.DeployTrack(authCollector,client)
	if err!=nil{
	    log.Fatal(err)
	}
	client.Commit()

	instance,err:= import_contracts.NewTrack(contractAddress,client)
	if err != nil {
			log.Fatal(err)
	}


	gateTuples_2_gate:=initializeCircuitTuples_asAnArray_TwoGate()

	var big_array [][gateTupleSize]*big.Int



	for i:=0;i<totalNumberOfGates;i++{
		var temp [gateTupleSize]*big.Int
		for j:=0;j<gateTupleSize;j++{
			temp[j]=big.NewInt(int64(gateTuples_2_gate[i][j]))
		}
		big_array=append(big_array,temp)
	}



	_,err=instance.InitializeCircuit(&bind.TransactOpts{
									From:authCollector.From,
									Signer:authCollector.Signer,
	},big_array,big.NewInt(maxLinesToGate))

	if err!=nil{
			log.Fatal(err)
	}

	client.Commit()


	channel_OwnerToCollector:=make(chan []byte)

	var wg sync.WaitGroup

	wg.Add(2)
	go Owner(authOwner,client,channel_OwnerToCollector,&wg)
	go Collector(authCollector,client,channel_OwnerToCollector,&wg)


	wg.Wait()  //Wait blocks until the WaitGroup counter is zero

}


// merkletree_enc_fileChunks:=createMerkleTreeForEncInp(encrypted_2_gate)
// complain,decodedOutputs,_,_:=Extract(encrypted_2_gate,seller_key,gateTuples_2_gate,merkletree_enc_fileChunks,merkleroot_encFile,merkleroot_file)
//
// fmt.Println("\n\n decodedOutputs : \n",decodedOutputs,"\n complain : ",complain)



func Owner(authOwner *bind.TransactOpts,client *backends.SimulatedBackend,channel_OwnerToCollector chan []byte,wg *sync.WaitGroup){

	prefix:=strconv.FormatUint(uint64(buffer_size),10)+" "

	owner_round1_computingTime_calculatingVariables_start := time.Now()

	fileChunks := FileRead()

	var OwnerToContractObject OwnerToContractStruct
	var OwnerToCollectorObject OwnerToCollectorStruct

	gateTuples_gates_2:=initializeCircuitTuples_asAnArray_TwoGate()

	key:= keyGenerate()
	keyCommit:=fnKeycommit(key)
	merkleroot_file:=createMerkleRootForInputVectors(fileChunks)
	encodedGateOutputs,out:=Encode(fileChunks,key,gateTuples_gates_2,merkleroot_file)
	merkleroot_encFile:=createMerkleRootForEncInp(encodedGateOutputs)

	fmt.Println("\nOut : \n",out)
	fmt.Println("\nencodedGateOutputs : \n",encodedGateOutputs,"\n\nmerkleroot_encFile :\n",merkleroot_encFile)
	fmt.Println("\n merkleroot_file : \n",merkleroot_file)

	setPrice(&OwnerToContractObject)
	OwnerToContractObject.Keycommit=keyCommit
	OwnerToContractObject.MerkleRootOfEncInp=merkleroot_encFile
	OwnerToContractObject.MerkleRootOfFile=merkleroot_file


	owner_round1_computingTime_calculatingVariables_stop:=time.Now()
	log.SetPrefix(prefix)
	log.Printf("onwer_round1_computingTime_calculatingVariables_time : %v \n", owner_round1_computingTime_calculatingVariables_stop.Sub(owner_round1_computingTime_calculatingVariables_start))

	var err error
	var tx *types.Transaction

	instance,err:= import_contracts.NewTrack(contractAddress,client)
	if err != nil {
			log.Fatal(err)
	}

	owner_round1_commitTime_InitializingRootsAndPrice_start := time.Now()

	tx,err=instance.SellerInitializeRootHashFileAndEncFile(&bind.TransactOpts{
									From:authOwner.From,
									Signer:authOwner.Signer,
	},OwnerToContractObject.MerkleRootOfFile,OwnerToContractObject.MerkleRootOfEncInp,OwnerToContractObject.Keycommit,&OwnerToContractObject.Price)
	if err!=nil{
			log.Fatal(err)
	}

	client.Commit()

	owner_round1_commitTime_InitializingRootsAndPrice_stop := time.Now()
	log.SetPrefix(prefix)
	log.Printf("owner_round1_commitTime_InitializingRootsAndPrice_time : %v \n", owner_round1_commitTime_InitializingRootsAndPrice_stop.Sub(owner_round1_commitTime_InitializingRootsAndPrice_start))
	log.SetPrefix(prefix)
	log.Println("Owner Round 1 : ",tx.Gas())



	OwnerToCollectorObject.EncryptedOutputOfGates=encodedGateOutputs
	OwnerToCollectorObject.OwnerAddress=authOwner.From


	owner_round1_communicationTime_sendingEncryptedGateOutputs_start := time.Now()

	byteArrayFromOnwerToCollector,err := json.Marshal(OwnerToCollectorObject)
	if err != nil {
		log.Fatal(err)
	}

	channel_OwnerToCollector<-byteArrayFromOnwerToCollector

	owner_round1_communicationTime_sendingEncryptedGateOutputs_stop := time.Now()


	log.SetPrefix(prefix)
	log.Printf("owner_round1_communicationTime_sendingEncryptedGateOutputs_time : %v \n", owner_round1_communicationTime_sendingEncryptedGateOutputs_stop.Sub(owner_round1_communicationTime_sendingEncryptedGateOutputs_start))


	onwer_round2_waitingTime_forCollectorToAccept_start := time.Now()


	ownerAcceptedStatus:=false
	for i:=0;i<waitingTime;i++ {

		MapAddressOwnerCollector_Obj,err:=instance.MapAddressBuyerSeller(nil,collectorAddress,authOwner.From)
		Phase:=MapAddressOwnerCollector_Obj.Phase
		// fmt.Println("\n Phase : ",Phase)
		if err!=nil{
			log.Fatal(err)
		}
		if(Phase==1){
			ownerAcceptedStatus=true
			break;
		}
		time.Sleep(time.Second *1)
	}

	owner_round2_waitingTime_forCollectorToAccept_stop := time.Now()
	log.SetPrefix(prefix)
	log.Printf("owner_round2_waitingTime_forCollectorToAccept_time : %v \n", owner_round2_waitingTime_forCollectorToAccept_stop.Sub(onwer_round2_waitingTime_forCollectorToAccept_start))


	if(!ownerAcceptedStatus){
			//Receiver has not accepted within time
			}else{

							owner_round3_commitTime_revealKey_start := time.Now()
							tx,err=instance.RevealKey(&bind.TransactOpts{
															From:authOwner.From,
															Signer:authOwner.Signer,
							},key,collectorAddress)

							if err!=nil{
									log.Fatal(err)
							}
							client.Commit()

							owner_round3_commitTime_revealKey_stop := time.Now()
							log.SetPrefix(prefix)
							log.Printf("owner_round3_commitTime_revealKey_time : %v \n", owner_round3_commitTime_revealKey_stop.Sub(owner_round3_commitTime_revealKey_start))
							//client.Commit()
							log.SetPrefix(prefix)
							log.Println("Owner Round 3 : ",tx.Gas())



							owner_round4_waitingTime_forCollectorToApproveOrProduceComplain_start:=time.Now()

							for i:=0;i<waitingTime;i++{

								MapAddressOwnerCollector_Obj,err:=instance.MapAddressBuyerSeller(nil,collectorAddress,authOwner.From)
								Phase:=MapAddressOwnerCollector_Obj.Phase
									if err!=nil{
										log.Fatal(err)
									}
									if(Phase==3){
										break
									}
									time.Sleep(time.Second *1)
							}

							owner_round4_waitingTime_forCollectorToApproveOrProduceComplain_stop:=time.Now()
							log.SetPrefix(prefix)
							log.Printf("owner_round4_waitingTime_forCollectorToApproveOrProduceComplain_time : %v \n", owner_round4_waitingTime_forCollectorToApproveOrProduceComplain_stop.Sub(owner_round4_waitingTime_forCollectorToApproveOrProduceComplain_start))



						MapAddressOwnerCollector_Obj,err:=instance.MapAddressBuyerSeller(nil,collectorAddress,authOwner.From)
						Phase:=MapAddressOwnerCollector_Obj.Phase
						if err!=nil{
							log.Fatal(err)
						}

						if(Phase==2){

								MapAddressOwnerCollector_Obj,err:=instance.MapAddressBuyerSeller(nil,collectorAddress,authOwner.From)
								timeout:=MapAddressOwnerCollector_Obj.Timeout
								if err!=nil{
												log.Fatal(err)
								}


								for{
									_,err:=instance.Nowy(&bind.TransactOpts{
										 From:authOwner.From,
										 Signer:authOwner.Signer,
									 })
									 if err!=nil{
												 log.Fatal(err)
									 }
									 client.Commit()

									 Now,err:=instance.Now(nil)
									 if err!=nil{
												 log.Fatal(err)
									 }

									 if(Now.Cmp(timeout)==1){
											break
									 }

									 time.Sleep(time.Second*1)
								}


								owner_round4_commitTime_getEtherOnSuccesfulCompletionOfProtocol_start:=time.Now()
								tx,err=instance.SenderGetEtherReceiverNoComplainMessage(&bind.TransactOpts{
										From:authOwner.From,
										Signer:authOwner.Signer,
									},collectorAddress)
								if err!=nil{
											log.Fatal(err)
								}
								client.Commit()

								owner_round4_commitTime_getEtherOnSuccesfulCompletionOfProtocol_stop:=time.Now()
								log.SetPrefix(prefix)
								log.Println("Owner Round 4 ",tx.Gas(),"\n")
								log.SetPrefix(prefix)
								log.Printf("owner_round4_commitTime_getEtherOnSuccesfulCompletionOfProtocol_time : %v \n", owner_round4_commitTime_getEtherOnSuccesfulCompletionOfProtocol_stop.Sub(owner_round4_commitTime_getEtherOnSuccesfulCompletionOfProtocol_start))

							}
				}
				wg.Done()
}

func Collector(authCollector *bind.TransactOpts,client *backends.SimulatedBackend,channel_OwnerToCollector chan []byte,wg *sync.WaitGroup){

	prefix:=strconv.FormatUint(uint64(buffer_size),10)+" "

	collector_round1_waitingTime_forReceivingEncryptedGateOutputs_start := time.Now()

	byteArrayFromOwnerToCollector:=<-channel_OwnerToCollector

	collector_round1_waitingTime_forReceivingEncryptedGateOutputs_stop := time.Now()
	log.SetPrefix(prefix)
	log.Printf("receiver_round1_waitingTime_forReceivingEncryptedGateOutputs_time : %v \n", collector_round1_waitingTime_forReceivingEncryptedGateOutputs_stop.Sub(collector_round1_waitingTime_forReceivingEncryptedGateOutputs_start),"\n")



	collector_round1_computingTime_merkleRootCheck_start := time.Now()


	var Owner_OwnerToCollectorObj OwnerToCollectorStruct

	err:=json.Unmarshal(byteArrayFromOwnerToCollector, &Owner_OwnerToCollectorObj)
	if err != nil {
			log.Fatal(err)
	}


	instance,err:= import_contracts.NewTrack(contractAddress,client)
	if err != nil {
			log.Fatal(err)
	}

	Collector_EncryptedOutputOfGates:=Owner_OwnerToCollectorObj.EncryptedOutputOfGates
	Collector_Owners_Address:=Owner_OwnerToCollectorObj.OwnerAddress
	gateTuples_gates_2:=initializeCircuitTuples_asAnArray_TwoGate()

	MapSellerFileDetails_Object,err:=instance.MapSellerFileDetails(nil,Collector_Owners_Address)
	Collector_MerkleRootOfEncInp:=MapSellerFileDetails_Object.RootHashEncFile
	if err!=nil{
		log.Fatal(err)
	}

	boolValueEncryptedGateOutput:=checkMerkleRootForEncInp(Collector_EncryptedOutputOfGates,Collector_MerkleRootOfEncInp)

	collector_round1_computingTime_merkleRootCheck_stop := time.Now()
	log.SetPrefix(prefix)
	log.Printf("collector_round1_computingTime_merkleRootCheck_time : %v \n", collector_round1_computingTime_merkleRootCheck_stop.Sub(collector_round1_computingTime_merkleRootCheck_start))


		if(boolValueEncryptedGateOutput){

				 value:=big.NewInt(100)

				 collector_round2_commitTime_Accept_start := time.Now()

				 tx,err:=instance.BuyerAccept(&bind.TransactOpts{
												 From:authCollector.From,
												 Signer:authCollector.Signer,
												 Value:value,
				 },Collector_Owners_Address,Collector_MerkleRootOfEncInp)
				 if err!=nil{
						 log.Fatal(err)
				 }

				 client.Commit()

				collector_round2_commitTime_Accept_stop := time.Now()
				log.SetPrefix(prefix)
				log.Printf("collector_round2_commitTime_Accept_time : %v \n", collector_round2_commitTime_Accept_stop.Sub(collector_round2_commitTime_Accept_start))

				log.SetPrefix(prefix)
				log.Println("Collector Round 2 : ",tx.Gas())


				collector_round3_waitingTime_forReceiverToRevealKey_start := time.Now()
					ownerKeyRevealedStatus:=false
					protocolFinishedStatus:=false

					var contract_Key [keySize]byte

					for i:=0;i<waitingTime;i++ {

							MapAddressOwnerCollector_Obj,err:=instance.MapAddressBuyerSeller(nil,authCollector.From,Collector_Owners_Address)
							Phase:=MapAddressOwnerCollector_Obj.Phase
							if err!=nil{
								log.Fatal(err)
							}

							if(Phase==2){
								MapAddressOwnerFileDetails_Obj,err:=instance.MapSellerFileDetails(nil,Collector_Owners_Address)
								contract_Key=MapAddressOwnerFileDetails_Obj.Key
								if err!=nil{
									log.Fatal(err)
								}
								ownerKeyRevealedStatus=true
								break;
							}

							if(Phase==3){
								protocolFinishedStatus=true
								break;
							}

							time.Sleep(time.Second *1)

					}
					collector_round3_waitingTime_forCollecrorToRevealKey_stop := time.Now()
					log.SetPrefix(prefix)
					log.Printf("collector_round3_waitingTime_forCollecrorToRevealKey_time : %v \n", collector_round3_waitingTime_forCollecrorToRevealKey_stop.Sub(collector_round3_waitingTime_forReceiverToRevealKey_start))



					if(protocolFinishedStatus){
						wg.Done()
					}

					if(!ownerKeyRevealedStatus){

						MapAddressOwnerCollector_Obj,err:=instance.MapAddressBuyerSeller(nil,authCollector.From,Collector_Owners_Address)
						timeout:=MapAddressOwnerCollector_Obj.Timeout
						 if err!=nil{
									 log.Fatal(err)
						 }

						 for{

								_,err:=instance.Nowy(&bind.TransactOpts{
									From:authCollector.From,
									Signer:authCollector.Signer,
								})
								if err!=nil{
											log.Fatal(err)
								}
								client.Commit()
								Now,err:=instance.Now(nil)
								if err!=nil{
											log.Fatal(err)
								}
								if(Now.Cmp(timeout)==1){
									 break
									}
								time.Sleep(time.Second*1)
						 }

						 collector_round3_commitTime_OwnerKeyNotRevealedGettingEther_start := time.Now()
							//the receiver getting the ether
							tx,err=instance.ReceiverGetEtherSenderNotRevealedKey(&bind.TransactOpts{
									From:authCollector.From,
									Signer:authCollector.Signer,
									Value: nil,
								},Collector_Owners_Address)
							if err!=nil{
										log.Fatal(err)
							}
							client.Commit()

							collector_round3_commitTime_OwnerKeyNotRevealedGettingEther_stop := time.Now()
							log.SetPrefix(prefix)

							log.Println("Collector Round 3 (Sender Not Revealed Key) : ",tx.Cost())
							log.Printf("\ncollector_round3_commitTime_OwnerKeyNotRevealedGettingEther_stop : %v \n", collector_round3_commitTime_OwnerKeyNotRevealedGettingEther_stop.Sub(collector_round3_commitTime_OwnerKeyNotRevealedGettingEther_start))


					} else{

						collector_round4_computingTime_extraction_start := time.Now()

						MapAddressOwnerFileDetails_Obj,err:=instance.MapSellerFileDetails(nil,Collector_Owners_Address)
						if err!=nil{
									log.Fatal(err)
						}
						Collector_MerkleRootOfFileChunks:=MapAddressOwnerFileDetails_Obj.RootHashFile


						merkletree_enc_fileChunks:=createMerkleTreeForEncInp(Collector_EncryptedOutputOfGates)
						complain,decodedOutputs,encodedVectors_ThisGate,gateIndex:=Extract(Collector_EncryptedOutputOfGates,contract_Key,gateTuples_gates_2,merkletree_enc_fileChunks,Collector_MerkleRootOfEncInp,Collector_MerkleRootOfFileChunks)

						fmt.Println("\n decoded Outputs : \n",decodedOutputs,"\n compalin : \n",complain)

						collector_round4_computingTime_extraction_stop := time.Now()
						log.SetPrefix(prefix)
						log.Printf("collector_round4_computingTime_extraction_time : %v \n", collector_round4_computingTime_extraction_stop.Sub(collector_round4_computingTime_extraction_start))


						if(complain==nil){


						} else{

						collector_round4_commitTime_complain_start := time.Now()
							tx,err=instance.Complain(&bind.TransactOpts{
									From:authCollector.From,
									Signer:authCollector.Signer,
							},complain,encodedVectors_ThisGate,big.NewInt(int64(gateIndex)),Collector_Owners_Address)
						if err!=nil{
							log.Fatal(err)
						}
						client.Commit()

						collector_round4_commitTime_complain_stop := time.Now()
						log.SetPrefix(prefix)
						log.Printf("\ncollector_round4_commitTime_complain_stop : %v \n", collector_round4_commitTime_complain_stop.Sub(collector_round4_commitTime_complain_start))

						log.SetPrefix(prefix)
						log.Println("Collector Round 4 : ",tx.Cost())

							}

						}

		}	else{

					if(!boolValueEncryptedGateOutput){
							//log.Println("\n->Receiver\nThe calculated merkle root of encrypted gate outputs and merkle root of encrypted gate outputs got from the contract didn't match")
						} else{
								//log.Println("\n->Receiver\nThe calculated merkle root of gate tuples and merkle root of egate tuples got from the contract didn't match")
						}
		}

		wg.Done()

}

func setPrice(object *OwnerToContractStruct){

	object.Price=*big.NewInt(50)     //800 wei
}


func Extract(encodedGateOutputs [][]byte,key [32]byte,gateTuples [][gateTupleSize]byte,merkleTreeForEncGateOutput [][hashFunctionOutputBitSize]byte,Receiver_MerkleRootOfEncInp [hashFunctionOutputBitSize]byte,Receiver_MerkleRootOfFileChunks [hashFunctionOutputBitSize]byte) ([][][hashFunctionOutputBitSize]byte,[][]byte,[][]byte,int) {

	// var decodedOutputs [totalNumberOfGates][]byte
	decodedOutputs:=make([][]byte,totalNumberOfGates)
	var encodedVectors_ThisGate  [][]byte
	 //to store all the decodedOutputs. Dynamic in size due to the fact that a wrong gate operation can lead to stopping the decoding process and
	//generating the complaint
	var complain [][][hashFunctionOutputBitSize]byte
	var out []byte

	//Decryption of input vectors. Have taken the consumption that an error cannot be generated in the input gates. Even if the sender has something fishy in the
	//input gates, this effect will be refelected in the next level of gates.So, a relevant complaint can be given to the contract.
	for i:=0;i<noOfInputGates;i++ {
			out =Decrypt(key,encodedGateOutputs[i])
			for j:=0;j<len(out);j++{
				decodedOutputs[i] = append(decodedOutputs[i],out[j])
			}
	}

	//Decryption of not input gates - gates above the base level.
	for i:=noOfInputGates;i<totalNumberOfGates;i++{

		out = Decrypt(key,encodedGateOutputs[i])

		for j:=0;j<len(out);j++{
			decodedOutputs[i] = append(decodedOutputs[i],out[j])
		}

		operation:=int(gateTuples[i][1])

		var operationInputs [][]byte

		for k:=2;k<2+maxLinesToGate;k++{

			if(int(gateTuples[i][k])==totalNumberOfGates){
				break
			}
			operationInputs=append(operationInputs,decodedOutputs[gateTuples[i][k]])
		}
				 if(!reflect.DeepEqual(decodedOutputs[i],Operation(operation,operationInputs,decodedOutputs,Receiver_MerkleRootOfEncInp,Receiver_MerkleRootOfFileChunks)))	 {
				// OperationVal:=Operation(operation,operationInputs,decodedOutputs,Receiver_MerkleRootOfEncInp)
				// fmt.Println("OperationVal : ",OperationVal)
						encodedVectors_ThisGate=append(encodedVectors_ThisGate,encodedGateOutputs[i])
						//merkle proof for the output of a complaint gate
						MproofTree:=Mproof(i,merkleTreeForEncGateOutput)
						complain=append(complain,MproofTree)

						//merkle proof for the inputs of a complaint gate and appending to the complain vector
						for k:=2;k<2+maxLinesToGate;k++{

							if int(gateTuples[i][k])==totalNumberOfGates {
								break
							}
								encodedVectors_ThisGate=append(encodedVectors_ThisGate,encodedGateOutputs[(gateTuples[i][k])])
								MproofTree:=Mproof(int(gateTuples[i][k]),merkleTreeForEncGateOutput)
								complain=append(complain,MproofTree)
						}
						return complain,decodedOutputs,encodedVectors_ThisGate,i
					}
			}
	return complain,decodedOutputs,encodedVectors_ThisGate,-1
}

func Mproof(index int,merkletree [][hashFunctionOutputBitSize]byte) ([][hashFunctionOutputBitSize]byte){

	depth:=int(mathlib.Log2(float64(totalNumberOfGates)))+1
	tree:=make([][32]byte,depth)
	tree[0]=merkletree[index]
	for i:=1;i<depth;i++{
		if(index%2==0){
			tree[i]=merkletree[index+1]
			index=index/2+totalNumberOfGates
		} else {
			tree[i]=merkletree[index-1]
			index=index/2+totalNumberOfGates
		}
	}
	return tree

}

func Operation(Op int,operationInputs [][]byte,decodedOutputs [][]byte,Receiver_MerkleRootOfEncInp [hashFunctionOutputBitSize]byte,Receiver_MerkleRootOfFileChunks [hashFunctionOutputBitSize]byte)  ([]byte){

		var result []byte
		if(Op==2) {
				leftsibling:=operationInputs[0]
				rightsibling:=operationInputs[1]
				var toBeHashed []byte
				for j:=0;j<len(leftsibling);j++ {
						toBeHashed= append(toBeHashed,leftsibling[j])
				}
				for j:=0;j<len(rightsibling);j++ {
						toBeHashed= append(toBeHashed,rightsibling[j])
				}

				hashed := kec.Keccak256(toBeHashed)

				for j:=0;j<hashFunctionOutputBitSize;j++{
					result=append(result,hashed[j])
				}
			}

			if(Op==4) {
					toBeHashed:=operationInputs[0]
					hashed := kec.Keccak256(toBeHashed)

					for j:=0;j<hashFunctionOutputBitSize;j++{
						if (j==0){
								if(hashed[j]==Receiver_MerkleRootOfFileChunks[j]){
									result=append(result,1)
								} else {
									result=append(result,0)
											 }
								} else {
									if(hashed[j]==Receiver_MerkleRootOfFileChunks[j]){
										result=append(result,0)
									}	else{
										result=append(result,1)
									}
						}
				}
			}

		//Operation 3: The gate checks the equivalency of the MRx(input parameter) and the input to the gate.
		if(Op==3) {
			merkleTreeOfDecryptedInputVectors:=make([][]byte,noOfInputGates)
			for k:=0;k<noOfInputGates;k++{
						merkleTreeOfDecryptedInputVectors[k]=decodedOutputs[k]
			}
			merkleRootofDecodedInputVectors:=createMerkleRootForInputVectors(merkleTreeOfDecryptedInputVectors)
			for j:=0;j<hashFunctionOutputBitSize;j++{
				if (j==0){
						if(operationInputs[0][j]==merkleRootofDecodedInputVectors[j]){
							result=append(result,1)
						} else {
							result=append(result,0)
									 }
						} else {
							if(operationInputs[0][j]==merkleRootofDecodedInputVectors[j]){
								result=append(result,0)
							}	else{
								result=append(result,1)
							}
				}
			}
		}
		return result
}


func createMerkleTreeForEncInp(encryptedGateOutputs [][]byte) ([][hashFunctionOutputBitSize]byte){

	merkletree:=make([][hashFunctionOutputBitSize]byte,2*totalNumberOfGates-1)

	//input gates
	//no hashing of the input vectors to the fn.
	for i:=0;i<totalNumberOfGates;i++{
				merkletree[i]=Hash(encryptedGateOutputs[i])
	}

	//!inputgates
	//merkle tree construction
	k:=0
	for i:=0;i<totalNumberOfGates-1;i++{

		var toBeHashed []byte
		var Hashed []byte
		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k][j])
		}

		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k+1][j])
		}

		Hashed=kec.Keccak256(toBeHashed) //Hashing of concatenated input

		for j:=0;j<hashFunctionOutputBitSize;j++{
			merkletree[i+totalNumberOfGates][j]=Hashed[j]
		}
		k=k+2
	}

	  //returns the merkleroot
	return merkletree
}

func fnKeycommit(key [keySize]byte) ([hashFunctionOutputBitSize]byte){

			var toBEHashedKey []byte
			var HashedKey [hashFunctionOutputBitSize]byte

			for j:=0;j<keySize;j++{
				toBEHashedKey=append(toBEHashedKey,key[j])
			}

			HashedAfter:=kec.Keccak256(toBEHashedKey)

			for j:=0;j<hashFunctionOutputBitSize;j++{
					HashedKey[j]=HashedAfter[j]
			}

			return HashedKey
}



func Encode(inputVectors [][]byte,key [keySize]byte,gateTuples [][gateTupleSize]byte,MRx [hashFunctionOutputBitSize]byte) ([][]byte,[][]byte)  {

	out:=make([][]byte,totalNumberOfGates)
	z:=make([][]byte,totalNumberOfGates)

	for i:=0;i<noOfInputGates;i++ {
				out[i]=inputVectors[i]
				temp:=Encrypt(key,out[i])
				z[i]=temp
	}


	// fmt.Println("loser")

	//Not Input Gates
	//For each gate in this range,the input vectors to a particular gate are processed according to the operation and the ouput and encryption of this output is stored
	for i:=noOfInputGates;i<noOfInputGates+noOfInputGates/2;i++{


			Op:=int(gateTuples[i][1])  //Operation of gate by the index "i"

			//Operation 2: The gate takes in the output of two of its children, concatenates the two inputs and hashes it.Later the output is encrypted.
			if(Op==2) {
					leftsibling:=int(gateTuples[i][2])
					rightsibling:=int(gateTuples[i][3])
					var toBeHashed []byte
					for j:=0;j<buffer_size;j++ {
							toBeHashed= append(toBeHashed,out[leftsibling][j])
					}
					for j:=0;j<buffer_size;j++ {
							toBeHashed= append(toBeHashed,out[rightsibling][j])
					}
					hash := kec.Keccak256(toBeHashed)
					out[i]=hash
			}

			//Operation 3: The gate checks the equivalency of the MRx(input parameter) and the input to the gate.
			if(Op==3) {
				for j:=0;j<hashFunctionOutputBitSize;j++{
					if (j==0){
							if(out[i-1][j]==MRx[j]){
								out[i]=append(out[i],1)
							} else {
								out[i]=append(out[i],0)
										 }
							} else {
								if(out[i-1][j]==MRx[j]){
									out[i]=append(out[i],0)
								}	else{
								out[i]=append(out[i],1)
								}
					}
				}
			}

			if(Op==1) {
					for j:=0;j<buffer_size;j++{
						out[i]=append(out[i],byte(0))
					}
			}

			temp:=Encrypt(key,out[i])
			z[i]=temp
		}


	for i:=noOfInputGates+noOfInputGates/2;i<totalNumberOfGates;i++{


			Op:=int(gateTuples[i][1])  //Operation of gate by the index "i"

			//Operation 2: The gate takes in the output of two of its children, concatenates the two inputs and hashes it.Later the output is encrypted.
			if(Op==2) {
					leftsibling:=int(gateTuples[i][2])
					rightsibling:=int(gateTuples[i][3])
					var toBeHashed []byte
					for j:=0;j<hashFunctionOutputBitSize;j++ {
							toBeHashed= append(toBeHashed,out[leftsibling][j])
					}
					for j:=0;j<hashFunctionOutputBitSize;j++ {
							toBeHashed= append(toBeHashed,out[rightsibling][j])
					}
					hash := kec.Keccak256(toBeHashed)
					for j:=0;j<hashFunctionOutputBitSize;j++{
						out[i]=append(out[i],hash[j])
					}
			}

			//Operation 3: The gate checks the equivalency of the MRx(input parameter) and the input to the gate.
			if(Op==3) {
				for j:=0;j<hashFunctionOutputBitSize;j++{
					if (j==0){
							if(out[i-1][j]==MRx[j]){
								out[i]=append(out[i],1)
							} else {
								out[i]=append(out[i],0)
										 }
							} else {
								if(out[i-1][j]==MRx[j]){
									out[i]=append(out[i],0)
								}	else{
								out[i]=append(out[i],1)
								}
					}
				}
			}


			if(Op==4) {
				hashed:=Hash(out[i-1])
				for j:=0;j<hashFunctionOutputBitSize;j++{
					if (j==0){
							if(hashed[j]==MRx[j]){
								out[i]=append(out[i],1)
							} else {
								out[i]=append(out[i],0)
										 }
							} else {
								if(hashed[j]==MRx[j]){
									out[i]=append(out[i],0)
								}	else{
								out[i]=append(out[i],1)
								}
					}
				}
			}

			if(Op==1) {
					for j:=0;j<hashFunctionOutputBitSize;j++{
						out[i]=append(out[i],byte(0))
					}
			}

			temp:=Encrypt(key,out[i])
			z[i]=temp
		}

	return z,out
}


func initializeCircuitTuples_asAnArray() ([][gateTupleSize]byte){


			var gateTuples [][gateTupleSize]byte
			gateTuples = make([][gateTupleSize]byte,totalNumberOfGates)

			paddingNumber:=totalNumberOfGates
			//inputGates
			for i:=0; i<noOfInputGates; i++ {
				gateTuples[i][0]=byte(i)
				gateTuples[i][1]=1
				noOfInputsToTheGate:=0
					for j:=2; j<2+noOfInputsToTheGate; j++ {

					}
					for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
						gateTuples[i][k]=byte(paddingNumber)
					}
			}

			//middleGates
			l:=0
			for i:=noOfInputGates; i<totalNumberOfGates-1; i++ {
				gateTuples[i][0]=byte(i)
				gateTuples[i][1]=2
				noOfInputsToTheGate:=2
					for j:=2; j<2+noOfInputsToTheGate; j++ {
							gateTuples[i][j]=byte(l)
							l++
					}
					for k:=2+noOfInputsToTheGate;  k<gateTupleSize; k++ {
						gateTuples[i][k]=byte(paddingNumber)
					}
			}

			//Lastgate
			gateTuples[totalNumberOfGates-1][0]=byte(totalNumberOfGates-1)
			gateTuples[totalNumberOfGates-1][1]=3
			noOfInputsToTheGate:=1;
				for j:=2; j<2+noOfInputsToTheGate; j++ {
						gateTuples[totalNumberOfGates-1][j]=byte(totalNumberOfGates-2)
				}
				for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
					gateTuples[totalNumberOfGates-1][k]=byte(paddingNumber)
				}

			return gateTuples
}

func initializeCircuitTuples_asAnArray_TwoGate() ([][gateTupleSize]byte){


			var gateTuples [][gateTupleSize]byte
			gateTuples = make([][gateTupleSize]byte,totalNumberOfGates)

			paddingNumber:=2
			//inputGates
			for i:=0; i<1; i++ {
				gateTuples[i][0]=byte(i)
				gateTuples[i][1]=1
				noOfInputsToTheGate:=0
					for j:=2; j<2+noOfInputsToTheGate; j++ {

					}
					for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
						gateTuples[i][k]=byte(paddingNumber)
					}
			}


			//Lastgate
			gateTuples[1][0]=byte(totalNumberOfGates-1)
			gateTuples[1][1]=4
			noOfInputsToTheGate:=1;
				for j:=2; j<2+noOfInputsToTheGate; j++ {
						gateTuples[1][j]=byte(0)
				}
				for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
					gateTuples[1][k]=byte(paddingNumber)
				}

			return gateTuples
}

//
// func mal_initializeCircuitTuples_asAnArray_TwoGate() ([][gateTupleSize]byte){
//
//
// 			var gateTuples [][gateTupleSize]byte
// 			gateTuples = make([][gateTupleSize]byte,totalNumberOfGates)
//
// 			paddingNumber:=2
// 			//inputGates
// 			for i:=0; i<1; i++ {
// 				gateTuples[i][0]=byte(i)
// 				gateTuples[i][1]=1
// 				noOfInputsToTheGate:=0
// 					for j:=2; j<2+noOfInputsToTheGate; j++ {
//
// 					}
// 					for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
// 						gateTuples[i][k]=byte(paddingNumber)
// 					}
// 			}
//
//
// 			//Lastgate
// 			gateTuples[1][0]=byte(totalNumberOfGates-1)
// 			gateTuples[1][1]=4
// 			noOfInputsToTheGate:=1;
// 				for j:=2; j<2+noOfInputsToTheGate; j++ {
// 						gateTuples[1][j]=byte(0)
// 				}
// 				for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
// 					gateTuples[1][k]=byte(paddingNumber)
// 				}
//
// 			return gateTuples
// }


func keyGenerate() ([keySize]byte){

	keyGen := make([]byte, keySize)
	var key [keySize]byte

  _, err := rand.Read(keyGen)
  if err != nil {
      log.Fatal(err)
  }

	for i:=0;i<len(keyGen);i++{
		key[i]=keyGen[i]
	}
	return key
}


func maliciousInitializeCircuitTuples_asAnArray() ([][gateTupleSize]byte){


			var gateTuples [][gateTupleSize]byte
			gateTuples = make([][gateTupleSize]byte,totalNumberOfGates)

			paddingNumber:=totalNumberOfGates
			//inputGates
			for i:=0; i<noOfInputGates; i++ {
				gateTuples[i][0]=byte(i)
				gateTuples[i][1]=1
				noOfInputsToTheGate:=0
					for j:=2; j<2+noOfInputsToTheGate; j++ {

					}
					for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
						gateTuples[i][k]=byte(paddingNumber)
					}
			}

			//middleGates
			l:=0
			for i:=noOfInputGates; i<totalNumberOfGates-1; i++ {
				gateTuples[i][0]=byte(i)
				gateTuples[i][1]=2
				noOfInputsToTheGate:=2
					for j:=2; j<2+noOfInputsToTheGate; j++ {
							gateTuples[i][j]=byte(l)
							l++
					}
					for k:=2+noOfInputsToTheGate;  k<gateTupleSize; k++ {
						gateTuples[i][k]=byte(paddingNumber)
					}
			}

			//Lastgate
			gateTuples[totalNumberOfGates-1][0]=byte(totalNumberOfGates-1)
			gateTuples[totalNumberOfGates-1][1]=3
			noOfInputsToTheGate:=1;
				for j:=2; j<2+noOfInputsToTheGate; j++ {
						gateTuples[totalNumberOfGates-1][j]=byte(totalNumberOfGates-2)
				}
				for k:=2+noOfInputsToTheGate; k<gateTupleSize; k++ {
					gateTuples[totalNumberOfGates-1][k]=byte(paddingNumber)
				}

			gateTuples[noOfInputGates][1]=1

			return gateTuples
}


func generateFinalKey(key [keySize]byte,keySizeNeeded int) ([]byte) {
	var keyHashed []byte
	var temp [32]byte
	var finalKey []byte
	finalKey = make([]byte, keySizeNeeded)

	for i:=0;i<keySizeNeeded/keySize;i++ {

		if(i==0){
			var keyPlusIndex []byte

			k:=0
			for j:=0;j<keySize;j++{
				keyPlusIndex=append(keyPlusIndex,key[j])
			}
			keyPlusIndex=append(keyPlusIndex,byte(k))
			keyHashed = kec.Keccak256(keyPlusIndex)

			for j:=0;j<hashFunctionOutputBitSize;j++{
				temp[j]=keyHashed[j]
				finalKey[j]=temp[j]
			}

		}else{

			keyHashed=nil

			for j:=0;j<hashFunctionOutputBitSize;j++{
				keyHashed=append(keyHashed,temp[j])
			}
			keyHashed=kec.Keccak256(keyHashed)

			for j:=0;j<hashFunctionOutputBitSize;j++{
				temp[j]=keyHashed[j]
				finalKey[i*32+j]=temp[j]
			}
		}
	}
	return finalKey
}

func Encrypt(key [keySize]byte,plainText []byte) ([]byte) {

			finalKey:=generateFinalKey(key,len(plainText))
			var encryptedtext []byte
			for i:=0;i<len(plainText);i++{
				encryptedtext=append(encryptedtext,finalKey[i]^plainText[i])
			}
			return encryptedtext
}

func Decrypt(key [keySize]byte,encryptedtext []byte) ([]byte){

			finalKey:=generateFinalKey(key,len(encryptedtext))
			var plainText []byte
			for i:=0;i<len(encryptedtext);i++{
				plainText=append(plainText,finalKey[i]^encryptedtext[i])
			}

			return plainText
}

func Hash(_toBeHashed []byte) ([32]byte){

	var toBeHashed []byte
	var Hashed [hashFunctionOutputBitSize]byte

	for j:=0;j<len(_toBeHashed);j++{
		toBeHashed=append(toBeHashed,_toBeHashed[j])
	}

	HashedAfter:=kec.Keccak256(toBeHashed)

	for j:=0;j<hashFunctionOutputBitSize;j++{
			Hashed[j]=HashedAfter[j]
	}

	return Hashed
}


func AuthAndAddressGeneration() (*bind.TransactOpts,common.Address){

	privateKey,err:=crypto.GenerateKey()
  if err!=nil{
    log.Fatal(err)
  }
  auth:=bind.NewKeyedTransactor(privateKey)

  publicKey := privateKey.Public()
  publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
  if !ok {
      log.Fatal("error casting public key to ECDSA")
  }
  address := crypto.PubkeyToAddress(*publicKeyECDSA)
  return auth,address
}


func FileRead() ([][]byte){
	var currentByte int64 = 0
	fileBuffer := make([]byte, buffer_size)
	FileChunks := make([][]byte, noOfInputGates)
	file, err := os.Open("a.txt")
	if err != nil {
			log.Fatal(err)
	}
	count:=0
	for {
			n,err := file.ReadAt(fileBuffer, currentByte)
			if err!=nil{
				log.Println(err)
			}
			currentByte += int64(buffer_size)
			fileBuffer=fileBuffer[:n]
			if count==noOfInputGates{
					break
			}
			FileChunks[count] = make([]byte, buffer_size)
			copy(FileChunks[count],fileBuffer)
			if err == io.EOF {
					log.Fatal(err)
			}
			count++
	}
	file.Close()
	return FileChunks
}

func checkMerkleRootForEncInp(encryptedGateOutputs [][]byte,merklerootforEncInput [hashFunctionOutputBitSize]byte) (bool){

	// var merkletree [2*totalNumberOfGates-1][hashFunctionOutputBitSize]byte
	merkletree:=make([][hashFunctionOutputBitSize]byte,2*totalNumberOfGates-1)

	//input gates
	//no hashing of the input vectors to the fn.
	for i:=0;i<totalNumberOfGates;i++{
				merkletree[i]=Hash(encryptedGateOutputs[i])
	}

	//!inputgates
	//merkle tree construction
	k:=0
	for i:=0;i<totalNumberOfGates-1;i++{

		var toBeHashed []byte
		var Hashed []byte
		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k][j])
		}

		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k+1][j])
		}

		Hashed=kec.Keccak256(toBeHashed) //Hashing of concatenated input
		for j:=0;j<hashFunctionOutputBitSize;j++{
			merkletree[i+totalNumberOfGates][j]=Hashed[j]
		}
		k=k+2
	}

	for j:=0;j<hashFunctionOutputBitSize;j++{
		if(merkletree[2*totalNumberOfGates-2][j]!=merklerootforEncInput[j]){
			return false
		}
	}

	return true

}

//
//
func createMerkleRootForInputVectors(inputVectors [][]byte) ([hashFunctionOutputBitSize]byte){

	if(len(inputVectors)==1) {
		return(Hash(inputVectors[0]))
	}

	var merkletree [][]byte
	merkletree = make([][]byte,2*noOfInputGates-1)

	//input gates
	for i:=0;i<noOfInputGates;i++{
				merkletree[i]=inputVectors[i]
	}

	//!inputgates
	k:=0

	for i:=noOfInputGates;i<noOfInputGates+noOfInputGates/2;i++{

		var toBeHashed []byte
		var Hashed []byte

		for j:=0;j<buffer_size;j++{
			toBeHashed=append(toBeHashed,merkletree[k][j])
		}

		for j:=0;j<buffer_size;j++{
			toBeHashed=append(toBeHashed,merkletree[k+1][j])
		}

		Hashed=kec.Keccak256(toBeHashed)

		for j:=0;j<hashFunctionOutputBitSize;j++{
			merkletree[i]=append(merkletree[i],Hashed[j])
		}

		k=k+2

	}


	for i:=noOfInputGates+noOfInputGates/2;i<2*noOfInputGates-1;i++{

		var toBeHashed []byte
		var Hashed []byte

		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k][j])
		}

		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k+1][j])
		}

		Hashed=kec.Keccak256(toBeHashed)

		for j:=0;j<hashFunctionOutputBitSize;j++{
			merkletree[i]=append(merkletree[i],Hashed[j])
		}
		k=k+2
	}


	var merkleRoot [hashFunctionOutputBitSize]byte
	for j:=0;j<hashFunctionOutputBitSize;j++{
		merkleRoot[j]=merkletree[2*noOfInputGates-2][j]
	}
	return merkleRoot
}

func createMerkleRootForEncInp(encryptedGateOutputs [][]byte) ([hashFunctionOutputBitSize]byte){

	// var merkletree [2*totalNumberOfGates-1][hashFunctionOutputBitSize]byte
	merkletree:=make([][hashFunctionOutputBitSize]byte,2*totalNumberOfGates-1)

	//input gates
	//no hashing of the input vectors to the fn.
	for i:=0;i<totalNumberOfGates;i++{
				merkletree[i]=Hash(encryptedGateOutputs[i])
	}

	//!inputgates
	//merkle tree construction
	k:=0
	for i:=0;i<totalNumberOfGates-1;i++{

		var toBeHashed []byte
		var Hashed []byte
		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k][j])
		}

		for j:=0;j<hashFunctionOutputBitSize;j++{
			toBeHashed=append(toBeHashed,merkletree[k+1][j])
		}

		Hashed=kec.Keccak256(toBeHashed) //Hashing of concatenated input
		for j:=0;j<hashFunctionOutputBitSize;j++{
			merkletree[i+totalNumberOfGates][j]=Hashed[j]
		}
		k=k+2
	}

	  //returns the merkleroot
	return merkletree[2*totalNumberOfGates-2]
}
//
//
//
// func Encode(inputVectors [][]byte,key [32]byte,MRx [hashFunctionOutputBitSize]byte) ([][]byte,[][]byte)  {
//
// 	out:=make([][]byte,totalNumberOfGates)
// 	z:=make([][]byte,totalNumberOfGates)
//
// 	for i:=0;i<noOfInputGates;i++ {
// 			out[i]=inputVectors[i]
// 			temp:=Enc(key,out[i])
// 			z[i]=temp
// 	}
//
// 	//Not Input Gates
// 	//For each gate in this range,the input vectors to a particular gate are processed according to the operation and the ouput and encryption of this output is stored
// 	l:=0
// 	for i:=noOfInputGates;i<noOfInputGates+noOfInputGates/2;i++{
// 			Op:=2  //Operation of gate by the index "i"
//
// 			//Operation 2: The gate takes in the output of two of its children, concatenates the two inputs and hashes it.Later the output is encrypted.
//
// 			if(Op==2) {
// 					leftsibling:=l
// 					l++
// 					rightsibling:=l
// 					l++
// 					var toBeHashed []byte
// 					for j:=0;j<buffer_size;j++ {
// 							toBeHashed= append(toBeHashed,out[leftsibling][j])
// 					}
// 					for j:=0;j<buffer_size;j++ {
// 							toBeHashed= append(toBeHashed,out[rightsibling][j])
// 					}
// 					hash := kec.Keccak256(toBeHashed)
// 					out[i]=hash
// 			}
//
//
// 			temp:=Enc(key,out[i])
// 			z[i]=temp
//
// 			if(i==8){
// 				fmt.Println("\n\ninside encode =",out[i],"\n\nenc : ",z[i])
// 			}
// 		}
//
//
// 	for i:=noOfInputGates+noOfInputGates/2;i<totalNumberOfGates;i++{
// 			Op:=2  //Operation of gate by the index "i"
// 			if(i==totalNumberOfGates-1){
// 				Op=3
// 			}
//
// 			//Operation 2: The gate takes in the output of two of its children, concatenates the two inputs and hashes it.Later the output is encrypted.
// 			if(Op==2) {
// 					leftsibling:=l
// 					l++
// 					rightsibling:=l
// 					l++
// 					var toBeHashed []byte
// 					for j:=0;j<hashFunctionOutputBitSize;j++ {
// 							toBeHashed= append(toBeHashed,out[leftsibling][j])
// 					}
// 					for j:=0;j<hashFunctionOutputBitSize;j++ {
// 							toBeHashed= append(toBeHashed,out[rightsibling][j])
// 					}
// 					hash := kec.Keccak256(toBeHashed)
// 					for j:=0;j<hashFunctionOutputBitSize;j++{
// 						out[i]=append(out[i],hash[j])
// 					}
// 			}
//
// 			//Operation 3: The gate checks the equivalency of the MRx(input parameter) and the input to the gate.
// 			if(Op==3) {
// 				for j:=0;j<hashFunctionOutputBitSize;j++{
// 					if (j==0){
// 							if(out[i-1][j]==MRx[j]){
// 								out[i]=append(out[i],1)
// 							} else {
// 								out[i]=append(out[i],0)
// 										 }
// 							} else {
// 								if(out[i-1][j]==MRx[j]){
// 									out[i]=append(out[i],0)
// 								}	else{
// 								out[i]=append(out[i],1)
// 								}
// 					}
// 				}
// 			}
//
// 			if(Op==1) {
// 					for j:=0;j<hashFunctionOutputBitSize;j++{
// 						out[i]=append(out[i],byte(0))
// 					}
// 			}
//
// 			temp:=Enc(key,out[i])
// 			z[i]=temp
// 		}
//
// 	return z,out
// }
//
// func checkMerkleRootForEncInp(encryptedGateOutputs [][]byte,merklerootforEncInput [hashFunctionOutputBitSize]byte) (bool){
//
// 	// var merkletree [2*totalNumberOfGates-1][hashFunctionOutputBitSize]byte
// 	merkletree:=make([][hashFunctionOutputBitSize]byte,2*totalNumberOfGates-1)
//
// 	//input gates
// 	//no hashing of the input vectors to the fn.
// 	for i:=0;i<totalNumberOfGates;i++{
// 				merkletree[i]=Hash(encryptedGateOutputs[i])
// 	}
//
// 	//!inputgates
// 	//merkle tree construction
// 	k:=0
// 	for i:=0;i<totalNumberOfGates-1;i++{
//
// 		var toBeHashed []byte
// 		var Hashed []byte
// 		for j:=0;j<hashFunctionOutputBitSize;j++{
// 			toBeHashed=append(toBeHashed,merkletree[k][j])
// 		}
//
// 		for j:=0;j<hashFunctionOutputBitSize;j++{
// 			toBeHashed=append(toBeHashed,merkletree[k+1][j])
// 		}
//
// 		Hashed=kec.Keccak256(toBeHashed) //Hashing of concatenated input
// 		for j:=0;j<hashFunctionOutputBitSize;j++{
// 			merkletree[i+totalNumberOfGates][j]=Hashed[j]
// 		}
// 		k=k+2
// 	}
//
// 	for j:=0;j<hashFunctionOutputBitSize;j++{
// 		if(merkletree[2*totalNumberOfGates-2][j]!=merklerootforEncInput[j]){
// 			return false
// 		}
// 	}
//
// 	return true
//
// }
//
//
// func Extract(encodedGateOutputs [][]byte,key [32]byte,merkleTreeForEncGateOutput [][hashFunctionOutputBitSize]byte,Receiver_MerkleRootOfEncInp [hashFunctionOutputBitSize]byte,Receiver_MerkleRootOfFile [hashFunctionOutputBitSize]byte) ([][][hashFunctionOutputBitSize]byte,[][]byte,[][]byte) {
//
// 	// var decodedOutputs [totalNumberOfGates][]byte
// 	decodedOutputs:=make([][]byte,totalNumberOfGates)
// 	var encodedVectors_ThisGate  [][]byte
// 	 //to store all the decodedOutputs. Dynamic in size due to the fact that a wrong gate operation can lead to stopping the decoding process and
// 	//generating the complaint
// 	var complain [][][hashFunctionOutputBitSize]byte
// 	var out []byte
//
// 	//Decryption of input vectors. Have taken the consumption that an error cannot be generated in the input gates. Even if the sender has something fishy in the
// 	//input gates, this effect will be refelected in the next level of gates.So, a relevant complaint can be given to the contract.
// 	for i:=0;i<noOfInputGates;i++ {
// 			out =Decrypt(key,encodedGateOutputs[i])
// 			for j:=0;j<len(out);j++{
// 				decodedOutputs[i] = append(decodedOutputs[i],out[j])
// 			}
// 	}
//
// 	//Decryption of not input gates - gates above the base level.
// 	l:=0
// 	for i:=noOfInputGates;i<totalNumberOfGates;i++{
//
// 		Op:=2  //Operation of gate by the index "i"
// 		if(i==totalNumberOfGates-1){
// 			Op=3
// 		}
//
// 		out = Decrypt(key,encodedGateOutputs[i])
//
// 		for j:=0;j<len(out);j++{
// 			decodedOutputs[i] = append(decodedOutputs[i],out[j])
// 		}
//
// 		var operationInputs [][]byte
// 		operationInputs=append(operationInputs,decodedOutputs[l])
// 		operationInputs=append(operationInputs,decodedOutputs[l+1])
//
// 		// fmt.Println("\n\ndecoded Output : ",decodedOutputs[i],"\nOperationValue : ",Operation(operationInputs,decodedOutputs))
//
// 				 if(!reflect.DeepEqual(decodedOutputs[i],Operation(Op,operationInputs,decodedOutputs,Receiver_MerkleRootOfFile)))	 {
// 				// OperationVal:=Operation(operation,operationInputs,decodedOutputs,Receiver_MerkleRootOfEncInp)
// 				// fmt.Println("OperationVal : ",OperationVal)
// 						fmt.Println("\n\n Complain Index : ",i)
// 						encodedVectors_ThisGate=append(encodedVectors_ThisGate,encodedGateOutputs[i])
//
// 						MproofTree:=Mproof(i,merkleTreeForEncGateOutput)
// 						complain=append(complain,MproofTree)
//
// 						//merkle proof for the inputs of a complaint gate and appending to the complain vector
// 						for k:=0;k<2;k++{
//
// 								encodedVectors_ThisGate=append(encodedVectors_ThisGate,encodedGateOutputs[l+k])
// 								MproofTree:=Mproof(l+k,merkleTreeForEncGateOutput)
// 								complain=append(complain,MproofTree)
// 						}
// 						return complain,decodedOutputs,encodedVectors_ThisGate
// 					}
// 					l=l+2
// 			}
//
// 	return complain,decodedOutputs,encodedVectors_ThisGate
// }
//
// func Mproof(index int,merkletree [][hashFunctionOutputBitSize]byte) ([][hashFunctionOutputBitSize]byte){
//
// 	depth:=int(mathlib.Log2(float64(totalNumberOfGates)))+1
// 	tree:=make([][32]byte,depth)
// 	tree[0]=merkletree[index]
// 	for i:=1;i<depth;i++{
// 		if(index%2==0){
// 			tree[i]=merkletree[index+1]
// 			index=index/2+totalNumberOfGates
// 		} else {
// 			tree[i]=merkletree[index-1]
// 			index=index/2+totalNumberOfGates
// 		}
// 	}
// 	return tree
//
// }
//
//
// func Operation(operation int,operationInputs [][]byte,decodedOutputs [][]byte,merkleRootOfFile [hashFunctionOutputBitSize]byte)  ([]byte){
//
// 		var result []byte
// 		if(operation==2){
// 				leftsibling:=operationInputs[0]
// 				rightsibling:=operationInputs[1]
// 				var toBeHashed []byte
// 				for j:=0;j<len(leftsibling);j++ {
// 						toBeHashed= append(toBeHashed,leftsibling[j])
// 				}
// 				for j:=0;j<len(rightsibling);j++ {
// 						toBeHashed= append(toBeHashed,rightsibling[j])
// 				}
//
// 				hashed := kec.Keccak256(toBeHashed)
//
// 				for j:=0;j<hashFunctionOutputBitSize;j++{
// 					result=append(result,hashed[j])
// 				}
// 		}
//
// 		if(operation==3){
// 			for j:=0;j<hashFunctionOutputBitSize;j++{
// 				if (j==0){
// 						if(operationInputs[0][j]==merkleRootOfFile[j]){
// 							result=append(result,1)
// 						} else {
// 							result=append(result,0)
// 									 }
// 						} else {
// 							if(operationInputs[0][j]==merkleRootOfFile[j]){
// 								result=append(result,0)
// 							}	else{
// 								result=append(result,1)
// 							}
// 				}
// 			}
// 		}
//
// 		return result
// }
//
// func Mverify(complaint [][32]byte,_index int,ciphertextRoot [32]byte) (bool){
//
// 	_value:=complaint[0]
// 	depth:=uint(len(complaint))
// 	var Hashed []byte
// 	var i uint
// 	for i=0; i<depth-1; i++{
// 		if ((_index & (1<<i))>>i == 1)	{
//
// 			var toBeHashed []byte
// 			for j:=0;j<32;j++{
// 				toBeHashed=append(toBeHashed,complaint[i+1][j])
// 			}
// 			for j:=0;j<32;j++{
// 				toBeHashed=append(toBeHashed, _value[j])
//
// 			}
//
// 			Hashed = kec.Keccak256(toBeHashed)
//
// 			for j:=0;j<32;j++{
// 				_value[j]=Hashed[j]
//
// 			}
//
// 		}else{
//
// 			var toBeHashed []byte
// 			for j:=0;j<32;j++{
// 				toBeHashed=append(toBeHashed,_value[j])
// 			}
// 			for j:=0;j<32;j++{
// 				toBeHashed=append(toBeHashed,complaint[i+1][j])
// 			}
//
// 			Hashed = kec.Keccak256(toBeHashed)
//
// 			for j:=0;j<32;j++{
// 				_value[j]=Hashed[j]
// 			}
//
// 		}
//
// 	}
// 	return (_value==ciphertextRoot)
// }
//
//
// func createMerkleTreeForEncInp(encryptedGateOutputs [][]byte) ([][hashFunctionOutputBitSize]byte){
//
// 	merkletree:=make([][hashFunctionOutputBitSize]byte,2*totalNumberOfGates-1)
//
// 	//input gates
// 	//no hashing of the input vectors to the fn.
// 	for i:=0;i<totalNumberOfGates;i++{
// 				merkletree[i]=Hash(encryptedGateOutputs[i])
// 	}
//
// 	//!inputgates
// 	//merkle tree construction
// 	k:=0
// 	for i:=0;i<totalNumberOfGates-1;i++{
//
// 		var toBeHashed []byte
// 		var Hashed []byte
// 		for j:=0;j<hashFunctionOutputBitSize;j++{
// 			toBeHashed=append(toBeHashed,merkletree[k][j])
// 		}
//
// 		for j:=0;j<hashFunctionOutputBitSize;j++{
// 			toBeHashed=append(toBeHashed,merkletree[k+1][j])
// 		}
//
// 		Hashed=kec.Keccak256(toBeHashed) //Hashing of concatenated input
//
// 		for j:=0;j<hashFunctionOutputBitSize;j++{
// 			merkletree[i+totalNumberOfGates][j]=Hashed[j]
// 		}
// 		k=k+2
// 	}
//
// 	  //returns the merkleroot
// 	return merkletree
// }
//
//
// func Decrypt(key [32]byte,encryptedtext []byte) ([]byte){
//
// 			finalKey:=generateFinalKey(key,len(encryptedtext))
// 			var plainText []byte
// 			for i:=0;i<len(encryptedtext);i++{
// 				plainText=append(plainText,finalKey[i]^encryptedtext[i])
// 			}
//
// 			return plainText
// }
