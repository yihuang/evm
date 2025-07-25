/// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "../common/Types.sol";

/// @dev The IGov contract's address.
address constant GOV_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000805;

/// @dev The IGov contract's instance.
IGov constant GOV_CONTRACT = IGov(GOV_PRECOMPILE_ADDRESS);

/**
 * @dev VoteOption enumerates the valid vote options for a given governance proposal.
 */
enum VoteOption {
    // Unspecified defines a no-op vote option.
    Unspecified,
    // Yes defines a yes vote option.
    Yes,
    // Abstain defines an abstain vote option.
    Abstain,
    // No defines a no vote option.
    No,
    // NoWithVeto defines a no with veto vote option.
    NoWithVeto
}
/// @dev WeightedVote represents a vote on a governance proposal
struct WeightedVote {
    uint64 proposalId;
    address voter;
    WeightedVoteOption[] options;
    string metadata;
}

/// @dev WeightedVoteOption represents a weighted vote option
struct WeightedVoteOption {
    VoteOption option;
    string weight;
}

/// @dev DepositData represents information about a deposit on a proposal
struct DepositData {
    uint64 proposalId;
    address depositor;
    Coin[] amount;
}

/// @dev TallyResultData represents the tally result of a proposal
struct TallyResultData {
    string yes;
    string abstain;
    string no;
    string noWithVeto;
}

/// @dev ProposalData represents a governance proposal
struct ProposalData {
    uint64 id;
    string[] messages;
    uint32 status;
    TallyResultData finalTallyResult;
    uint64 submitTime;
    uint64 depositEndTime;
    Coin[] totalDeposit;
    uint64 votingStartTime;
    uint64 votingEndTime;
    string metadata;
    string title;
    string summary;
    address proposer;
}

/// @dev Params defines the governance parameters
struct Params {
    int64 votingPeriod;
    Coin[] minDeposit;
    int64 maxDepositPeriod;
    string quorum;
    string threshold;
    string vetoThreshold;
    string minInitialDepositRatio;
    string proposalCancelRatio;
    string proposalCancelDest;
    int64 expeditedVotingPeriod;
    string expeditedThreshold;
    Coin[] expeditedMinDeposit;
    bool burnVoteQuorum;
    bool burnProposalDepositPrevote;
    bool burnVoteVeto;
    string minDepositRatio;
}

/// @author The Evmos Core Team
/// @title Gov Precompile Contract
/// @dev The interface through which solidity contracts will interact with Gov
interface IGov {
    /// @dev SubmitProposal defines an Event emitted when a proposal is submitted.
    /// @param proposer the address of the proposer
    /// @param proposalId the proposal of id
    event SubmitProposal(address indexed proposer, uint64 proposalId);

    /// @dev CancelProposal defines an Event emitted when a proposal is canceled.
    /// @param proposer the address of the proposer
    /// @param proposalId the proposal of id
    event CancelProposal(address indexed proposer, uint64 proposalId);

    /// @dev Deposit defines an Event emitted when a deposit is made.
    /// @param depositor the address of the depositor
    /// @param proposalId the proposal of id
    /// @param amount the amount of the deposit
    event Deposit(address indexed depositor, uint64 proposalId, Coin[] amount);

    /// @dev Vote defines an Event emitted when a proposal voted.
    /// @param voter the address of the voter
    /// @param proposalId the proposal of id
    /// @param option the option for voter
    event Vote(address indexed voter, uint64 proposalId, uint8 option);

    /// @dev VoteWeighted defines an Event emitted when a proposal voted.
    /// @param voter the address of the voter
    /// @param proposalId the proposal of id
    /// @param options the options for voter
    event VoteWeighted(
        address indexed voter,
        uint64 proposalId,
        WeightedVoteOption[] options
    );

    /// TRANSACTIONS

    /// @notice submitProposal creates a new proposal from a protoJSON document.
    /// @dev submitProposal defines a method to submit a proposal.
    /// @param jsonProposal The JSON proposal
    /// @param deposit The deposit for the proposal
    /// @return proposalId The proposal id
    function submitProposal(
        address proposer,
        bytes calldata jsonProposal,
        Coin[] calldata deposit
    ) external returns (uint64 proposalId);

    /// @dev cancelProposal defines a method to cancel a proposal.
    /// @param proposalId The proposal id
    /// @return success Whether the transaction was successful or not
    function cancelProposal(
        address proposer,
        uint64 proposalId
    ) external returns (bool success);

    /// @dev deposit defines a method to add a deposit to a proposal.
    /// @param proposalId The proposal id
    /// @param amount The amount to deposit
    function deposit(
        address depositor,
        uint64 proposalId,
        Coin[] calldata amount
    ) external returns (bool success);


    /// @dev vote defines a method to add a vote on a specific proposal.
    /// @param voter The address of the voter
    /// @param proposalId the proposal of id
    /// @param option the option for voter
    /// @param metadata the metadata for voter send
    /// @return success Whether the transaction was successful or not
    function vote(
        address voter,
        uint64 proposalId,
        VoteOption option,
        string memory metadata
    ) external returns (bool success);

    /// @dev voteWeighted defines a method to add a vote on a specific proposal.
    /// @param voter The address of the voter
    /// @param proposalId The proposal id
    /// @param options The options for voter
    /// @param metadata The metadata for voter send
    /// @return success Whether the transaction was successful or not
    function voteWeighted(
        address voter,
        uint64 proposalId,
        WeightedVoteOption[] calldata options,
        string memory metadata
    ) external returns (bool success);

    /// QUERIES

    /// @dev getVote returns the vote of a single voter for a
    /// given proposalId.
    /// @param proposalId The proposal id
    /// @param voter The voter on the proposal
    /// @return vote Voter's vote for the proposal
    function getVote(
        uint64 proposalId,
        address voter
    ) external view returns (WeightedVote memory vote);

    /// @dev getVotes Returns the votes for a specific proposal.
    /// @param proposalId The proposal id
    /// @param pagination The pagination options
    /// @return votes The votes for the proposal
    /// @return pageResponse The pagination information
    function getVotes(
        uint64 proposalId,
        PageRequest calldata pagination
    )
        external
        view
        returns (WeightedVote[] memory votes, PageResponse memory pageResponse);

    /// @dev getDeposit returns the deposit of a single depositor for a given proposalId.
    /// @param proposalId The proposal id
    /// @param depositor The address of the depositor
    /// @return deposit The deposit information
    function getDeposit(
        uint64 proposalId,
        address depositor
    ) external view returns (DepositData memory deposit);

    /// @dev getDeposits returns all deposits for a specific proposal.
    /// @param proposalId The proposal id
    /// @param pagination The pagination options
    /// @return deposits The deposits for the proposal
    /// @return pageResponse The pagination information
    function getDeposits(
        uint64 proposalId,
        PageRequest calldata pagination
    )
        external
        view
        returns (
            DepositData[] memory deposits,
            PageResponse memory pageResponse
        );

    /// @dev getTallyResult returns the tally result of a proposal.
    /// @param proposalId The proposal id
    /// @return tallyResult The tally result of the proposal
    function getTallyResult(
        uint64 proposalId
    ) external view returns (TallyResultData memory tallyResult);

    /// @dev getProposal returns the proposal details based on proposal id.
    /// @param proposalId The proposal id
    /// @return proposal The proposal data
    function getProposal(
        uint64 proposalId
    ) external view returns (ProposalData memory proposal);

    /// @dev getProposals returns proposals with matching status.
    /// @param proposalStatus The proposal status to filter by
    /// @param voter The voter address to filter by, if any
    /// @param depositor The depositor address to filter by, if any
    /// @param pagination The pagination config
    /// @return proposals The proposals matching the filter criteria
    /// @return pageResponse The pagination information
    function getProposals(
        uint32 proposalStatus,
        address voter,
        address depositor,
        PageRequest calldata pagination
    )
        external
        view
        returns (
            ProposalData[] memory proposals,
            PageResponse memory pageResponse
        );

    /// @dev getParams returns the current governance parameters.
    /// @return params The governance parameters
    function getParams() external view returns (Params memory params);

    /// @dev getConstitution returns the current constitution.
    /// @return constitution The current constitution
    function getConstitution() external view returns (string memory constitution);
}

