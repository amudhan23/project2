pragma solidity >0.4.0 <=0.6.0;
pragma experimental ABIEncoderV2;

contract track{

	//circuit gateTuples
	uint[32][] public circuit;
	uint public depth;
	uint public testIndex;
	uint public circuitLength;
	uint public totalNumberOfGates;
	uint public maxInputLinesToAGate;
	uint public Now; //to get the current time


	bytes32 public OperationValue;
	bytes32 public Out;

	//root hash of file and enc file
	bytes32 public rootHashFile;
	bytes32 public rootHashEncFile;


	//key
	bytes32 public commitmentOfKey;
	bytes32 public key;
	uint public price;

	//summa variables
	bool public validComplain;

	//parrty addresses


	enum stage{intialized,active,revealed,finalized}
	/* stage public phase = stage.created; */

	struct buyer_seller{

		address payable seller;
		address payable buyer;
		stage phase;
		uint timeout;

	}

	struct seller_file_details
	{
		bytes32 rootHashFile;
		bytes32 rootHashEncFile;
		bytes32 commitmentOfKey;
		bytes32 key;
		uint price;
	}

	function Nowy() public {
		Now=now;
	}

	function nextStage(address payable _buyer,address payable _seller) public{

		map_address_buyer_seller[_buyer][_seller].phase=stage(uint(map_address_buyer_seller[_buyer][_seller].phase)+1);
		map_address_buyer_seller[_buyer][_seller].timeout=now+ 40 seconds;

	}


	modifier allowedBefore(address payable _buyer,address payable _seller,stage _stage)
	{
			require(map_address_buyer_seller[_buyer][_seller].phase ==_stage);
			require(now<map_address_buyer_seller[_buyer][_seller].timeout);
			_;
	}

	modifier allowedAfter(address payable _buyer,address payable _seller,stage _stage)
	{
			require(map_address_buyer_seller[_buyer][_seller].phase ==_stage);
			require(now>map_address_buyer_seller[_buyer][_seller].timeout);
			_;
	}


	//seller to buyer to buyer_seller struct
	mapping(address => mapping(address => buyer_seller)) public map_address_buyer_seller;
	/* mapping(address => bytes32) public  map_seller_rootHashEncFile;
	mapping(address => bytes32) public map_buyer_rootHashFile; */

	mapping(address => seller_file_details) public map_seller_fileDetails;

	function initialize_circuit(uint[32][] memory _circuit,uint _maxInputLinesToAGate) public
	{
			circuitLength=_circuit.length;
			totalNumberOfGates=circuitLength;
			maxInputLinesToAGate=_maxInputLinesToAGate;
			circuit = new uint[32][](_circuit.length);
			circuit=_circuit;
			depth=log2(_circuit.length)+1;
			/* testIndex=circuit[5][1]; */
	}


	function seller_initialize_rootHash_file_And_EncFile(bytes32 _rootHashFile, bytes32 _rootHashEncFile,bytes32 _commitmentOfKey,uint _price) public
	{
			rootHashFile=_rootHashFile;
			rootHashEncFile=_rootHashEncFile;
			commitmentOfKey=_commitmentOfKey;

			seller_file_details memory temp;
			temp.rootHashFile = _rootHashFile;
			temp.rootHashEncFile = _rootHashEncFile;
			temp.commitmentOfKey = _commitmentOfKey;
			temp.price=_price;
			map_seller_fileDetails[msg.sender]=temp;
	}

	function buyer_accept(address payable _seller,bytes32 _rootHashEncFile) payable public
	{
			/* require(map_seller_fileDetails[_seller].rootHashFile==_rootHashFile); */
			require(map_seller_fileDetails[_seller].rootHashEncFile==_rootHashEncFile);
			if(msg.value < price)
			{
				msg.sender.transfer(msg.value);

			}
			else
			{
				buyer_seller memory temp;
				temp.seller=_seller;
				temp.buyer=msg.sender;
				temp.phase=stage.active;
				temp.timeout=now+40 seconds;
				map_address_buyer_seller[msg.sender][_seller]=temp;
			}
	}

	function revealKey (bytes32 _key, address payable _buyer) allowedBefore(_buyer,msg.sender,stage.active) public
	{
			//add modifier for checking the stage and timeline
			require(map_address_buyer_seller[_buyer][msg.sender].seller == msg.sender);
			bytes32 Hashed;
			bytes memory toBeHashed;
			toBeHashed=new bytes (32);
			for (uint i=0;i<32;i++)
			{
					toBeHashed[i]=_key[i];
			}
			Hashed=keccak256(toBeHashed);

			if(Hashed!=map_seller_fileDetails[msg.sender].commitmentOfKey){
				_buyer.transfer(map_seller_fileDetails[msg.sender].price);
			}
			else
			{
				map_seller_fileDetails[msg.sender].key=_key;
				/* map_address_buyer_seller[_buyer][msg.sender].phase=stage.revealed; */
				nextStage(_buyer,msg.sender);

			}
	}


	/* bytes32 public OperationValue;/ */

	function complain(bytes32[][] memory complaint,bytes[] memory encodedVectors_ThisGate,uint _gateIndex,address payable _seller)  allowedBefore(msg.sender,_seller,stage.revealed) public returns(bool)
	{

			require(map_address_buyer_seller[msg.sender][_seller].buyer == msg.sender);
			/* uint8 operation; */
			uint8[] memory indicesOfInputGates;
			uint8 noOFinputToThisParticularGate;
			bytes[] memory inputVectorsToTheGate;
			/* bytes32 Out;
			bytes memory Outy; */
			bool check;

			if(encodedVectors_ThisGate.length!=complaint.length)
			{
				_seller.transfer(price);
				return false;
			}


			for(uint i=0;i<encodedVectors_ThisGate.length;i++)
			{
				if(Hash(encodedVectors_ThisGate[i])!=complaint[i][0])
				{
					_seller.transfer(price);
					return false;
				}
			}

			if(_gateIndex>=circuit.length)
			{
				_seller.transfer(price);
				return false;
			}

			if(!Mverify(complaint[0],_gateIndex))
			{
				_seller.transfer(price);
				return false;
			}

			/* operation= */

			/* Outy=new bytes(encodedVectors_ThisGate[0].length);
			Outy=Dec(encodedVectors_ThisGate[0],map_seller_fileDetails[_seller].key); */

			for(uint i=2;i<2+maxInputLinesToAGate;i++)    //noOFinputToThisParticularGate
			{
				if(uint8(circuit[_gateIndex][i])==totalNumberOfGates){   //totalNumberOfGates
				  break;
				}
				noOFinputToThisParticularGate++;
			}

			indicesOfInputGates=new uint8[] (noOFinputToThisParticularGate);

			for(uint i=0;i<noOFinputToThisParticularGate;i++)   //noOFinputToThisParticularGate
			{
				indicesOfInputGates[i]=(uint8(circuit[_gateIndex][i+2]));
			}

			if(complaint.length-1!=noOFinputToThisParticularGate)
			{
				_seller.transfer(price);
			  return false;
			}


			//uptoComplaintLength=true;
			inputVectorsToTheGate = new bytes[] (complaint.length-1);

			for(uint i=0;i<indicesOfInputGates.length;i++)
			{
					if(!Mverify(complaint[i+1],indicesOfInputGates[i]))
					{
						_seller.transfer(price);
						return false;
					}
					inputVectorsToTheGate[i]=new bytes (encodedVectors_ThisGate[i+1].length);
					inputVectorsToTheGate[i]=Dec(encodedVectors_ThisGate[i+1],map_seller_fileDetails[_seller].key);
			}

			/* OperationValue=Operation(operation,inputVectorsToTheGate); */
			/* Out=bytesToBytes32(Outy); */

			check=complain_supplement(uint8(circuit[_gateIndex][1]),inputVectorsToTheGate,encodedVectors_ThisGate[0],_seller,map_seller_fileDetails[_seller].rootHashFile);

			if(!check)
			{
				validComplain=true;
				msg.sender.transfer(price);
				nextStage(msg.sender,_seller);
				return true;
			}

			_seller.transfer(price);
			nextStage(msg.sender,_seller);
			return false;

	}


	function complain_supplement(uint8 _operation,bytes[] memory inputVectorsToTheGate,bytes memory encodedVector,address payable _seller,bytes32 _root_HashFile) internal returns(bool)
	{
		/* bytes32 OperationValue; */
		bytes memory Outy;
		/* bytes32 Out; */

		OperationValue=Operation(_operation,inputVectorsToTheGate,_root_HashFile);


		Outy=new bytes(encodedVector.length);
		Outy=Dec(encodedVector,map_seller_fileDetails[_seller].key);
		Out=bytesToBytes32(Outy);

		if(OperationValue==Out)
		{
			return true;
		}
		else
		{
			return false;
		}


	}

	function Dec(bytes memory encryptedText,bytes32 key) internal pure returns(bytes memory)
	{

				bytes memory decoded;
				bytes memory finalKey;
				finalKey=new bytes (encryptedText.length);
				finalKey=generateKey(key,encryptedText.length);
				decoded=new bytes (encryptedText.length);
				for(uint i=0;i<encryptedText.length;i++){
					decoded[i]=finalKey[i]^encryptedText[i];
				}
				return decoded;
	}



	function generateKey(bytes32 key,uint keySize) internal pure  returns(bytes memory)
	{


			bytes memory finalKey;
			bytes memory keyHashed;
			bytes32 temp;
			finalKey=new bytes(keySize);
			keyHashed=new bytes(32);

			for(uint i=0;i<keySize/32;i++){

				if(i==0)
				{
					bytes memory keyPlusIndex;
					keyPlusIndex=new bytes (33);
					for(uint j=0;j<32;j++){
						keyPlusIndex[j]=key[j];
					}
					keyPlusIndex[32]=0;
					temp=keccak256(keyPlusIndex);

					for(uint j=0;j<32;j++)
					{
						keyHashed[j]=temp[j];
						finalKey[j]=temp[j];
					}
				}
				else
				{
					delete keyHashed;
					keyHashed=new bytes(32);

					for(uint j=0;j<32;j++)
					{
							keyHashed[j]=temp[j];
					}
					temp=keccak256(keyHashed);

					for(uint j=0;j<32;j++)
					{
						keyHashed[j]=temp[j];
						finalKey[i*32+j]=temp[j];
					}

				}

			}
			return finalKey;
	}



	function Operation(uint operation,bytes[] memory inputVectorsToTheGate,bytes32 _root_HashFile) internal  returns(bytes32)
	{
			bytes32 Hashed;
			bytes memory equality_bytes;
			bytes32 equality;

			if(operation==2)
			{
						bytes memory toBeHashed;

						toBeHashed=new bytes (2*inputVectorsToTheGate[0].length);
						for (uint i=0;i<inputVectorsToTheGate[0].length;i++)
						{
							toBeHashed[i]=inputVectorsToTheGate[0][i];
						}
						for (uint i=0;i<inputVectorsToTheGate[1].length;i++)
						{
							toBeHashed[i+inputVectorsToTheGate[0].length]=inputVectorsToTheGate[1][i];
						}
						Hashed=keccak256(toBeHashed);
						return Hashed;
			}


			if(operation==3)
			{
						bytes32 temp;
						temp=bytesToBytes32(inputVectorsToTheGate[0]);
						equality_bytes=new bytes (32);
						uint8 a=1;

						if(temp==rootHashFile)
						{
							equality_bytes[0]=byte(a);
							equality=bytesToBytes32(equality_bytes);
							return equality;
						}
						else
						{
							for(uint8 i=0;i<32;i++)
							{
								equality_bytes[i]=byte(a);
							}
							equality_bytes[0]=0;
							equality=bytesToBytes32(equality_bytes);
							return equality;
						}
			}

			if(operation==4)
			{
				bytes memory toBeHashed;
				uint8 a=1;
				equality_bytes=new bytes (32);

				toBeHashed=new bytes (inputVectorsToTheGate[0].length);
				for (uint i=0;i<inputVectorsToTheGate[0].length;i++)
				{
					toBeHashed[i]=inputVectorsToTheGate[0][i];
				}
				Hashed=keccak256(toBeHashed);

				if(Hashed==_root_HashFile)
				{
					equality_bytes[0]=byte(a);
					equality=bytesToBytes32(equality_bytes);
					return equality;
				}
				else
				{
					for(uint8 i=0;i<32;i++)
					{
						equality_bytes[i]=byte(a);
					}
					equality_bytes[0]=0;
					equality=bytesToBytes32(equality_bytes);
					return equality;
				}
			}

	}


	function Hash(bytes memory _toBeHashed) internal pure returns(bytes32)
	{
			bytes32 Hashed;
			bytes memory toBeHashed;
			toBeHashed=new bytes (_toBeHashed.length);
			for (uint i=0;i<_toBeHashed.length;i++)
			{
				toBeHashed[i]=_toBeHashed[i];
			}
			Hashed=keccak256(toBeHashed);
			return Hashed;
	}



	function  Mverify(bytes32[] memory complaint,uint _index)  internal view returns(bool)
	{


				bytes32 _value;
				_value=complaint[0];
				bytes32 Hashed;
				uint i ;
				bytes memory toBeHashed;
				toBeHashed=new bytes (64);

				for(i=0; i<depth-1; i++)
				{
					//if condition is used to decide whether to append the _value with the complaint vector or append complaint vector
					//with the _value
					if ((_index&(1<<i))>>i == 1)
					{

						for (uint j=0;j<32;j++)
						{
							toBeHashed[j]=complaint[i+1][j];
						}
						for (uint j=0;j<32;j++)
						{
							toBeHashed[j+32]=_value[j];
						}


						Hashed=keccak256(toBeHashed);
						_value=Hashed;
					}
					else
					{
						for (uint j=0;j<32;j++)
						{
							toBeHashed[j]=_value[j];
						}
						for (uint j=0;j<32;j++)
						{
							toBeHashed[j+32]=complaint[i+1][j];
						}

						Hashed=keccak256(toBeHashed);
						_value=Hashed;
					}
				}
				return (_value==rootHashEncFile);
	}


	function bytesToBytes32(bytes memory source) public returns (bytes32 result)
	{

		    bytes memory tempEmptyStringTest = bytes(source);
		    if (tempEmptyStringTest.length == 0) {
		        return 0x0;
		    }

		    assembly {
		        result := mload(add(source, 32))
		    }
	}


	function gotTheCorrectFiles(address payable _seller) allowedBefore(msg.sender,_seller,stage.revealed) public{
		require(map_address_buyer_seller[msg.sender][_seller].buyer == msg.sender);
		nextStage(msg.sender,_seller);
		_seller.transfer(price);
	}


	//function is called by receiver in the accepted stage if the receiver doesn't reveal the key within the time. The
	//ether is transferred to the receiver
	function receiverGetEther_SenderNotRevealedKey(address payable _seller) allowedAfter(msg.sender,_seller,stage.active) public{
		require(map_address_buyer_seller[msg.sender][_seller].buyer == msg.sender);
		msg.sender.transfer(price);
		map_address_buyer_seller[msg.sender][_seller].phase=stage.finalized;
	}

	function senderGetEther_ReceiverNoComplainMessage(address payable _buyer) allowedAfter(_buyer,msg.sender,stage.revealed) public{
		require(map_address_buyer_seller[_buyer][msg.sender].seller == msg.sender);
		msg.sender.transfer(price);
		map_address_buyer_seller[_buyer][msg.sender].phase=stage.finalized;
	}


	//log function
	function log2(uint x) private pure returns (uint y)
  {
     assembly
     {
		     let arg := x
		     x := sub(x,1)
		     x := or(x, div(x, 0x02))
		     x := or(x, div(x, 0x04))
		     x := or(x, div(x, 0x10))
		     x := or(x, div(x, 0x100))
		     x := or(x, div(x, 0x10000))
		     x := or(x, div(x, 0x100000000))
		     x := or(x, div(x, 0x10000000000000000))
		     x := or(x, div(x, 0x100000000000000000000000000000000))
		     x := add(x, 1)
		     let m := mload(0x40)
		     mstore(m,           0xf8f9cbfae6cc78fbefe7cdc3a1793dfcf4f0e8bbd8cec470b6a28a7a5a3e1efd)
		     mstore(add(m,0x20), 0xf5ecf1b3e9debc68e1d9cfabc5997135bfb7a7a3938b7b606b5b4b3f2f1f0ffe)
		     mstore(add(m,0x40), 0xf6e4ed9ff2d6b458eadcdf97bd91692de2d4da8fd2d0ac50c6ae9a8272523616)
		     mstore(add(m,0x60), 0xc8c0b887b0a8a4489c948c7f847c6125746c645c544c444038302820181008ff)
		     mstore(add(m,0x80), 0xf7cae577eec2a03cf3bad76fb589591debb2dd67e0aa9834bea6925f6a4a2e0e)
		     mstore(add(m,0xa0), 0xe39ed557db96902cd38ed14fad815115c786af479b7e83247363534337271707)
		     mstore(add(m,0xc0), 0xc976c13bb96e881cb166a933a55e490d9d56952b8d4e801485467d2362422606)
		     mstore(add(m,0xe0), 0x753a6d1b65325d0c552a4d1345224105391a310b29122104190a110309020100)
		     mstore(0x40, add(m, 0x100))
		     let magic := 0x818283848586878898a8b8c8d8e8f929395969799a9b9d9e9faaeb6bedeeff
		     let shift := 0x100000000000000000000000000000000000000000000000000000000000000
		     let a := div(mul(x, magic), shift)
		     y := div(mload(add(m,sub(255,a))), shift)
		     y := add(y, mul(256, gt(arg, 0x8000000000000000000000000000000000000000000000000000000000000000)))
    	}
	}











}
