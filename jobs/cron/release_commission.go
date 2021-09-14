package cron

import (
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/models"
)

type ReleaseCommissionJob struct {
}

func (j *ReleaseCommissionJob) Process() {
	s := gocron.NewScheduler()
	s.Every(1).Day().At("00:00:00").Do(func() {

	})
	s.Start()
}

type GroupReferral struct {
	FriendTrade uint64
	MemberID    uint64
}

type GroupUserReferral struct {
	Friend uint64
	UID    string
}

func (j *ReleaseCommissionJob) ReleaseReferrals() {
	var group_referrals []*GroupReferral

	yesterday := time.Now().Round(time.Hour*24).AddDate(0, 0, -1)

	config.DataBase.
		Model(&models.Commission{}).
		Select("COUNT(friend_uid) as friend_trade", "member_id").
		Where("created_at >= ?", yesterday).
		Group("member_id").
		Find(&group_referrals)

	for _, group_referral := range group_referrals {
		var referrals []*models.Commission

		earned_usdt := decimal.Zero

		config.DataBase.Where("member_id = ? created_at >= ?", group_referral.MemberID, yesterday).Find(&referrals)

		for _, referral := range referrals {
			var currency *models.Currency

			config.DataBase.First(&currency, "id = ?", referral.CurrencyID)
			earned_usdt = earned_usdt.Add(currency.Price.Mul(referral.EarnAmount))
		}

		var btc_currency *models.Currency
		config.DataBase.First(&btc_currency, "id = ?", "btc")

		earned_btc := earned_usdt.Div(btc_currency.Price)

		config.DataBase.Create(&models.ReleaseCommission{
			AccountType: "spot",
			MemberID:    group_referral.MemberID,
			EarnedBTC:   earned_btc,
			FriendTrade: group_referral.FriendTrade,
			Friend:      0,
		})
	}

	var group_user_referrals []*GroupUserReferral

	config.DataBase.
		Model(&models.Member{}).
		Select("COUNT(referral_uid) as friend", "referral_uid as uid").
		Where("created_at >= ?", yesterday).
		Find(&group_user_referrals)

	for _, group_user_referral := range group_user_referrals {
		var member *models.Member
		var release_referral *models.ReleaseCommission

		config.DataBase.Where("uid = ?", group_user_referral.UID).Find(&member)
		if result := config.DataBase.Where("member_id = ?", member.ID).Last(&release_referral); result.Error != nil {
			config.DataBase.Model(&release_referral).Update("friend", group_user_referral.Friend)
		} else {
			config.DataBase.Create(&models.ReleaseCommission{
				AccountType: "spot",
				MemberID:    member.ID,
				EarnedBTC:   decimal.Zero,
				FriendTrade: 0,
				Friend:      group_user_referral.Friend,
			})
		}
	}
}
