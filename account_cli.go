package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Account and profile operations",
	Long:  `Manage your Z.AI account information and profile details.`,
}

var accountInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get account information",
	Long:  `Get detailed account information including email, status, and balance.`,
	RunE:  runAccountInfo,
}

var accountStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get account status",
	Long:  `Get current account status and subscription details.`,
	RunE:  runAccountStatus,
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountInfoCmd)
	accountCmd.AddCommand(accountStatusCmd)

	accountInfoCmd.Flags().String("format", "table", "Output format (table, json)")
	accountStatusCmd.Flags().String("format", "table", "Output format (table, json)")
}

func runAccountInfo(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	fmt.Println("👤 Getting Account Information...")
	fmt.Print("Contacting Z.AI Account API...\n\n")

	info, err := apiClient.Account().GetAccountInfo(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	if !info.Success {
		return fmt.Errorf("API returned error: %s (code: %d)", info.Msg, info.Code)
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return outputJSON(info)
	}

	fmt.Println("📊 Account Information")
	fmt.Println("====================")

	if info.Data != nil {
		fmt.Printf("User ID: %s\n", info.Data.UserID)
		fmt.Printf("Email: %s\n", info.Data.Email)
		fmt.Printf("Account Type: %s\n", info.Data.AccountType)
		fmt.Printf("Status: %s\n", info.Data.Status)
		fmt.Printf("Verified: %t\n", info.Data.Verified)

		if info.Data.Balance > 0 || info.Data.Credit > 0 {
			fmt.Println("\n💰 Balance Information:")
			if info.Data.Balance > 0 {
				fmt.Printf("  Cash Balance: %.2f %s\n", info.Data.Balance, info.Data.Currency)
			}
			if info.Data.Credit > 0 {
				fmt.Printf("  Credit Balance: %.2f %s\n", info.Data.Credit, info.Data.Currency)
			}
		}

		if !info.Data.CreatedAt.IsZero() {
			fmt.Printf("Created: %s\n", info.Data.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println("No account data available")
		fmt.Println("💡 This endpoint may require different permissions or account type")
	}

	return nil
}

func runAccountStatus(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	fmt.Println("🔍 Getting Account Status...")
	fmt.Print("Contacting Z.AI Account API...\n\n")

	status, err := apiClient.Account().GetAccountStatus(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get account status: %w", err)
	}

	if !status.Success {
		return fmt.Errorf("API returned error: %s (code: %d)", status.Msg, status.Code)
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return outputJSON(status)
	}

	fmt.Println("📊 Account Status")
	fmt.Println("================")

	if status.Data != nil {
		fmt.Printf("Account ID: %s\n", status.Data.AccountID)
		fmt.Printf("Status: %s\n", status.Data.Status)
		fmt.Printf("Plan: %s\n", status.Data.Plan)
		fmt.Printf("Quota Status: %s\n", status.Data.QuotaStatus)
		fmt.Printf("Has Balance: %t\n", status.Data.HasBalance)

		if !status.Data.ExpiresAt.IsZero() {
			fmt.Printf("Expires: %s\n", status.Data.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println("No status data available")
		fmt.Println("💡 This endpoint may require different permissions or account type")
	}

	return nil
}
