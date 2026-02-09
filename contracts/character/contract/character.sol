// SPDX-License-Identifier: LGPL-3.0
pragma solidity ^0.4.18;

/// @title CharacterNFT - Text-to-character NFT with platform fees
/// @notice Facilitates minting and trading of user-generated characters on-chain.
/// The platform collects an upfront mint fee and a percentage of all secondary transactions.
contract CharacterNFT {

    // ──────────────────────────────────────────────
    //  Types
    // ──────────────────────────────────────────────

    /// Character lifecycle stages
    enum Stage { Text, Image, Model3D, Video, Licensed }

    struct Character {
        address creator;
        uint256 createdAt;
        Stage   stage;
        string  metadataURI;   // off-chain metadata (traits, images, models)
        bytes32 traitHash;     // keccak256 of the original text attributes
    }

    // ──────────────────────────────────────────────
    //  State
    // ──────────────────────────────────────────────

    address public platform;           // platform wallet (fee receiver)
    uint256 public mintFee;            // flat fee in wei charged on mint
    uint256 public transactionFeeBps;  // basis-points taken on every transfer (e.g. 250 = 2.5%)

    uint256 public nextTokenId;
    mapping(uint256 => Character) public characters;
    mapping(uint256 => address)   public ownerOf;
    mapping(uint256 => address)   public approvedFor;
    mapping(address => uint256)   public balanceOf;

    // ──────────────────────────────────────────────
    //  Events
    // ──────────────────────────────────────────────

    event CharacterMinted(uint256 indexed tokenId, address indexed creator, bytes32 traitHash, string metadataURI);
    event Transfer(uint256 indexed tokenId, address indexed from, address indexed to, uint256 price, uint256 platformCut);
    event StageAdvanced(uint256 indexed tokenId, uint8 newStage, string newMetadataURI);
    event MintFeeUpdated(uint256 newFee);
    event TransactionFeeUpdated(uint256 newFeeBps);

    // ──────────────────────────────────────────────
    //  Modifiers
    // ──────────────────────────────────────────────

    modifier onlyPlatform() {
        require(msg.sender == platform);
        _;
    }

    modifier onlyOwnerOf(uint256 _tokenId) {
        require(ownerOf[_tokenId] == msg.sender);
        _;
    }

    // ──────────────────────────────────────────────
    //  Constructor
    // ──────────────────────────────────────────────

    /// @param _mintFee           Initial flat mint fee in wei
    /// @param _transactionFeeBps Initial secondary-sale fee in basis points
    function CharacterNFT(uint256 _mintFee, uint256 _transactionFeeBps) public {
        require(_transactionFeeBps <= 10000);
        platform          = msg.sender;
        mintFee           = _mintFee;
        transactionFeeBps = _transactionFeeBps;
    }

    // ──────────────────────────────────────────────
    //  Minting
    // ──────────────────────────────────────────────

    /// @notice Mint a new character NFT.  Caller pays `mintFee`.
    /// @param _metadataURI  URI pointing to off-chain metadata (traits JSON, images, etc.)
    /// @param _traitHash    keccak256 of the raw text attributes provided by the user
    function mint(string _metadataURI, bytes32 _traitHash) public payable returns (uint256 tokenId) {
        require(msg.value >= mintFee);

        tokenId = nextTokenId;
        nextTokenId++;

        characters[tokenId] = Character({
            creator:     msg.sender,
            createdAt:   block.timestamp,
            stage:       Stage.Text,
            metadataURI: _metadataURI,
            traitHash:   _traitHash
        });

        ownerOf[tokenId]  = msg.sender;
        balanceOf[msg.sender]++;

        // Send mint fee to platform
        if (mintFee > 0) {
            platform.transfer(mintFee);
        }
        // Refund any excess
        if (msg.value > mintFee) {
            msg.sender.transfer(msg.value - mintFee);
        }

        CharacterMinted(tokenId, msg.sender, _traitHash, _metadataURI);
    }

    // ──────────────────────────────────────────────
    //  Transfers (secondary sales with platform cut)
    // ──────────────────────────────────────────────

    /// @notice Transfer a character. If value is sent, platform takes its cut.
    /// @param _tokenId Token to transfer
    /// @param _to      Recipient
    function transferFrom(uint256 _tokenId, address _to) public payable {
        require(ownerOf[_tokenId] == msg.sender || approvedFor[_tokenId] == msg.sender);
        require(_to != address(0));

        uint256 platformCut = 0;
        if (msg.value > 0 && transactionFeeBps > 0) {
            platformCut = (msg.value * transactionFeeBps) / 10000;
            platform.transfer(platformCut);
            // Remainder goes to the seller (current owner)
            address seller = ownerOf[_tokenId];
            seller.transfer(msg.value - platformCut);
        }

        _transfer(_tokenId, ownerOf[_tokenId], _to);
        Transfer(_tokenId, ownerOf[_tokenId], _to, msg.value, platformCut);
    }

    /// @notice Approve another address to transfer a specific token
    function approve(uint256 _tokenId, address _approved) public onlyOwnerOf(_tokenId) {
        approvedFor[_tokenId] = _approved;
    }

    function _transfer(uint256 _tokenId, address _from, address _to) internal {
        balanceOf[_from]--;
        balanceOf[_to]++;
        ownerOf[_tokenId] = _to;
        approvedFor[_tokenId] = address(0);
    }

    // ──────────────────────────────────────────────
    //  Character progression
    // ──────────────────────────────────────────────

    /// @notice Advance a character to the next pipeline stage and update metadata.
    /// Only the token owner can advance their character.
    /// @param _tokenId       Token to advance
    /// @param _newMetadataURI Updated metadata URI reflecting the new stage
    function advanceStage(uint256 _tokenId, string _newMetadataURI) public onlyOwnerOf(_tokenId) {
        Character storage c = characters[_tokenId];
        require(uint8(c.stage) < uint8(Stage.Licensed));

        c.stage       = Stage(uint8(c.stage) + 1);
        c.metadataURI = _newMetadataURI;

        StageAdvanced(_tokenId, uint8(c.stage), _newMetadataURI);
    }

    // ──────────────────────────────────────────────
    //  Platform admin
    // ──────────────────────────────────────────────

    function setMintFee(uint256 _newFee) public onlyPlatform {
        mintFee = _newFee;
        MintFeeUpdated(_newFee);
    }

    function setTransactionFee(uint256 _newFeeBps) public onlyPlatform {
        require(_newFeeBps <= 10000);
        transactionFeeBps = _newFeeBps;
        TransactionFeeUpdated(_newFeeBps);
    }

    function transferPlatform(address _newPlatform) public onlyPlatform {
        require(_newPlatform != address(0));
        platform = _newPlatform;
    }

    // ──────────────────────────────────────────────
    //  Views
    // ──────────────────────────────────────────────

    function getCharacter(uint256 _tokenId) public view
        returns (address creator, uint256 createdAt, uint8 stage, string metadataURI, bytes32 traitHash)
    {
        Character storage c = characters[_tokenId];
        return (c.creator, c.createdAt, uint8(c.stage), c.metadataURI, c.traitHash);
    }

    function totalSupply() public view returns (uint256) {
        return nextTokenId;
    }
}
