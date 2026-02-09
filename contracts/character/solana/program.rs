// CharacterNFT Solana Program (Anchor framework)
//
// This is the Solana-side equivalent of the Ethereum CharacterNFT contract.
// Same business logic: mint with platform fee, transfer with platform cut,
// and stage progression from Text → Image → 3D → Video → Licensed.
//
// Build with: anchor build
// Deploy with: anchor deploy
//
// NOTE: This source is included for reference and deployment.  The Go service
// interacts with the deployed program via RPC, not by importing this Rust code.

use anchor_lang::prelude::*;
use anchor_lang::system_program;

declare_id!("CharNFT11111111111111111111111111111111111");

#[program]
pub mod character_nft {
    use super::*;

    pub fn initialize(
        ctx: Context<Initialize>,
        mint_fee_lamports: u64,
        transaction_fee_bps: u16,
    ) -> Result<()> {
        require!(transaction_fee_bps <= 10000, CharError::FeeTooHigh);
        let state = &mut ctx.accounts.state;
        state.platform = ctx.accounts.platform.key();
        state.mint_fee_lamports = mint_fee_lamports;
        state.transaction_fee_bps = transaction_fee_bps;
        state.next_token_id = 0;
        Ok(())
    }

    pub fn mint(
        ctx: Context<MintCharacter>,
        metadata_uri: String,
        trait_hash: [u8; 32],
    ) -> Result<()> {
        let state = &mut ctx.accounts.state;

        // Transfer mint fee to platform
        if state.mint_fee_lamports > 0 {
            system_program::transfer(
                CpiContext::new(
                    ctx.accounts.system_program.to_account_info(),
                    system_program::Transfer {
                        from: ctx.accounts.creator.to_account_info(),
                        to: ctx.accounts.platform.to_account_info(),
                    },
                ),
                state.mint_fee_lamports,
            )?;
        }

        let character = &mut ctx.accounts.character;
        character.token_id = state.next_token_id;
        character.creator = ctx.accounts.creator.key();
        character.owner = ctx.accounts.creator.key();
        character.created_at = Clock::get()?.unix_timestamp;
        character.stage = 0; // StageText
        character.metadata_uri = metadata_uri;
        character.trait_hash = trait_hash;

        state.next_token_id += 1;

        emit!(CharacterMinted {
            token_id: character.token_id,
            creator: character.creator,
            trait_hash,
        });

        Ok(())
    }

    pub fn transfer_from(
        ctx: Context<TransferCharacter>,
        sale_price_lamports: u64,
    ) -> Result<()> {
        let character = &mut ctx.accounts.character;
        require!(character.owner == ctx.accounts.owner.key(), CharError::NotOwner);

        // Calculate and transfer platform cut
        if sale_price_lamports > 0 {
            let state = &ctx.accounts.state;
            let platform_cut = (sale_price_lamports as u128)
                .checked_mul(state.transaction_fee_bps as u128)
                .unwrap()
                .checked_div(10000)
                .unwrap() as u64;

            // Platform cut
            if platform_cut > 0 {
                system_program::transfer(
                    CpiContext::new(
                        ctx.accounts.system_program.to_account_info(),
                        system_program::Transfer {
                            from: ctx.accounts.recipient.to_account_info(),
                            to: ctx.accounts.platform.to_account_info(),
                        },
                    ),
                    platform_cut,
                )?;
            }

            // Seller proceeds
            let seller_proceeds = sale_price_lamports - platform_cut;
            if seller_proceeds > 0 {
                system_program::transfer(
                    CpiContext::new(
                        ctx.accounts.system_program.to_account_info(),
                        system_program::Transfer {
                            from: ctx.accounts.recipient.to_account_info(),
                            to: ctx.accounts.owner.to_account_info(),
                        },
                    ),
                    seller_proceeds,
                )?;
            }
        }

        character.owner = ctx.accounts.recipient.key();

        emit!(CharacterTransferred {
            token_id: character.token_id,
            from: ctx.accounts.owner.key(),
            to: ctx.accounts.recipient.key(),
            price: sale_price_lamports,
        });

        Ok(())
    }

    pub fn advance_stage(
        ctx: Context<AdvanceStage>,
        new_metadata_uri: String,
    ) -> Result<()> {
        let character = &mut ctx.accounts.character;
        require!(character.owner == ctx.accounts.owner.key(), CharError::NotOwner);
        require!(character.stage < 4, CharError::AlreadyLicensed); // 4 = Licensed

        character.stage += 1;
        character.metadata_uri = new_metadata_uri;

        emit!(StageAdvanced {
            token_id: character.token_id,
            new_stage: character.stage,
        });

        Ok(())
    }

    pub fn set_mint_fee(ctx: Context<AdminAction>, new_fee_lamports: u64) -> Result<()> {
        let state = &mut ctx.accounts.state;
        require!(state.platform == ctx.accounts.platform.key(), CharError::NotOwner);
        state.mint_fee_lamports = new_fee_lamports;
        Ok(())
    }

    pub fn set_transaction_fee(ctx: Context<AdminAction>, new_fee_bps: u16) -> Result<()> {
        let state = &mut ctx.accounts.state;
        require!(state.platform == ctx.accounts.platform.key(), CharError::NotOwner);
        require!(new_fee_bps <= 10000, CharError::FeeTooHigh);
        state.transaction_fee_bps = new_fee_bps;
        Ok(())
    }
}

// ── Account structs ──────────────────────────────────────────────

#[account]
pub struct ProgramState {
    pub platform: Pubkey,
    pub mint_fee_lamports: u64,
    pub transaction_fee_bps: u16,
    pub next_token_id: u64,
}

#[account]
pub struct Character {
    pub token_id: u64,
    pub creator: Pubkey,
    pub owner: Pubkey,
    pub created_at: i64,
    pub stage: u8,
    pub metadata_uri: String,
    pub trait_hash: [u8; 32],
}

// ── Context structs ──────────────────────────────────────────────

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(mut)]
    pub platform: Signer<'info>,
    #[account(init, payer = platform, space = 8 + 32 + 8 + 2 + 8)]
    pub state: Account<'info, ProgramState>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct MintCharacter<'info> {
    #[account(mut)]
    pub creator: Signer<'info>,
    #[account(mut)]
    pub state: Account<'info, ProgramState>,
    #[account(init, payer = creator, space = 8 + 8 + 32 + 32 + 8 + 1 + 4 + 256 + 32)]
    pub character: Account<'info, Character>,
    /// CHECK: validated by state.platform
    #[account(mut, constraint = platform.key() == state.platform)]
    pub platform: AccountInfo<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct TransferCharacter<'info> {
    #[account(mut)]
    pub owner: Signer<'info>,
    #[account(mut)]
    pub character: Account<'info, Character>,
    /// CHECK: recipient receives ownership
    #[account(mut)]
    pub recipient: AccountInfo<'info>,
    /// CHECK: validated by state.platform
    #[account(mut, constraint = platform.key() == state.platform)]
    pub platform: AccountInfo<'info>,
    pub state: Account<'info, ProgramState>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct AdvanceStage<'info> {
    pub owner: Signer<'info>,
    #[account(mut)]
    pub character: Account<'info, Character>,
}

#[derive(Accounts)]
pub struct AdminAction<'info> {
    pub platform: Signer<'info>,
    #[account(mut)]
    pub state: Account<'info, ProgramState>,
}

// ── Events ───────────────────────────────────────────────────────

#[event]
pub struct CharacterMinted {
    pub token_id: u64,
    pub creator: Pubkey,
    pub trait_hash: [u8; 32],
}

#[event]
pub struct CharacterTransferred {
    pub token_id: u64,
    pub from: Pubkey,
    pub to: Pubkey,
    pub price: u64,
}

#[event]
pub struct StageAdvanced {
    pub token_id: u64,
    pub new_stage: u8,
}

// ── Errors ───────────────────────────────────────────────────────

#[error_code]
pub enum CharError {
    #[msg("Character is already at the final stage")]
    AlreadyLicensed,
    #[msg("Only the owner can perform this action")]
    NotOwner,
    #[msg("Transaction fee exceeds 10000 bps")]
    FeeTooHigh,
    #[msg("Insufficient lamports for mint fee")]
    InsufficientFunds,
}
