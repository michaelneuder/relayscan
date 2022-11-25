// Package database exposes the postgres database
package database

import (
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DatabaseService struct {
	DB *sqlx.DB
}

func NewDatabaseService(dsn string) (*DatabaseService, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.DB.SetMaxOpenConns(50)
	db.DB.SetMaxIdleConns(10)
	db.DB.SetConnMaxIdleTime(0)

	if os.Getenv("PRINT_SCHEMA") == "1" {
		fmt.Println(schema)
	}

	if os.Getenv("DB_DONT_APPLY_SCHEMA") == "" {
		_, err = db.Exec(schema)
		if err != nil {
			return nil, err
		}
	}

	return &DatabaseService{
		DB: db,
	}, nil
}

func (s *DatabaseService) Close() error {
	return s.DB.Close()
}

func (s *DatabaseService) SaveSignedBuilderBid(entry SignedBuilderBidEntry) error {
	query := `INSERT INTO ` + TableSignedBuilderBid + `
		(relay, requested_at, received_at, duration_ms, slot, parent_hash, proposer_pubkey, pubkey, signature, value, fee_recipient, block_hash, block_number, gas_limit, gas_used, extra_data, epoch, timestamp, prev_randao) VALUES
		(:relay, :requested_at, :received_at, :duration_ms, :slot, :parent_hash, :proposer_pubkey, :pubkey, :signature, :value, :fee_recipient, :block_hash, :block_number, :gas_limit, :gas_used, :extra_data, :epoch, :timestamp, :prev_randao)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveBuilder(entry *BlockBuilderEntry) error {
	query := `INSERT INTO ` + TableBlockBuilder + ` (builder_pubkey, description) VALUES (:builder_pubkey, :description) ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIPayloadDelivered(entry *DataAPIPayloadDeliveredEntry) error {
	query := `INSERT INTO ` + TableDataAPIPayloadDelivered + `
		(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, extra_data) VALUES
		(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value_claimed_wei, :value_claimed_eth, :num_tx, :block_number, :extra_data)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIPayloadDeliveredBatch(entries []*DataAPIPayloadDeliveredEntry) error {
	if len(entries) == 0 {
		return nil
	}
	query := `INSERT INTO ` + TableDataAPIPayloadDelivered + `
	(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, extra_data) VALUES
	(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value_claimed_wei, :value_claimed_eth, :num_tx, :block_number, :extra_data)
	ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entries)
	return err
}

func (s *DatabaseService) GetDataAPILatestPayloadDelivered(relay string) (*DataAPIPayloadDeliveredEntry, error) {
	entry := new(DataAPIPayloadDeliveredEntry)
	query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, extra_data, slot_missed, value_check_ok, value_check_method, value_delivered_wei, value_delivered_eth, value_delivered_diff_wei, value_delivered_diff_eth, block_coinbase_addr, block_coinbase_is_proposer, coinbase_diff_wei, coinbase_diff_eth, found_onchain, notes FROM ` + TableDataAPIPayloadDelivered + ` WHERE relay=$1 ORDER BY slot DESC LIMIT 1`
	err := s.DB.Get(entry, query, relay)
	return entry, err
}

func (s *DatabaseService) SaveDataAPIBid(entry *DataAPIBuilderBidEntry) error {
	query := `INSERT INTO ` + TableDataAPIBuilderBid + `
		(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp) VALUES
		(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value, :num_tx, :block_number, :timestamp)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIBids(entries []*DataAPIBuilderBidEntry) error {
	if len(entries) == 0 {
		return nil
	}
	query := `INSERT INTO ` + TableDataAPIBuilderBid + `
	(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp) VALUES
	(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value, :num_tx, :block_number, :timestamp)
	ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entries)
	return err
}

func (s *DatabaseService) GetDataAPILatestBid(relay string) (*DataAPIBuilderBidEntry, error) {
	entry := new(DataAPIBuilderBidEntry)
	query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp FROM ` + TableDataAPIBuilderBid + ` WHERE relay=$1 ORDER BY slot DESC, timestamp DESC LIMIT 1`
	err := s.DB.Get(entry, query, relay)
	return entry, err
}

func (s *DatabaseService) GetTopRelays(since, until time.Time) (res []*TopRelayEntry, err error) {
	query := `SELECT relay, count(relay) as payloads FROM mainnet_data_api_payload_delivered WHERE inserted_at > $1 AND inserted_at < $2 GROUP BY relay ORDER BY payloads DESC;`
	err = s.DB.Select(&res, query, since.UTC(), until.UTC())
	return res, err
}

func (s *DatabaseService) GetTopBuilders(since, until time.Time, relay string) (res []*TopBuilderEntry, err error) {
	query := `SELECT extra_data, count(extra_data) as blocks FROM (
		SELECT distinct(slot), extra_data FROM mainnet_data_api_payload_delivered WHERE inserted_at > $1 AND inserted_at < $2`
	if relay != "" {
		query += ` AND relay = '` + relay + `'`
	}
	query += ` GROUP BY slot, extra_data
	) as x GROUP BY extra_data ORDER BY blocks DESC;`
	err = s.DB.Select(&res, query, since.UTC(), until.UTC())
	return res, err
}

func (s *DatabaseService) GetBuilderProfits(since, until time.Time) (res []*BuilderProfitEntry, err error) {
	query := `SELECT
		extra_data,
		count(extra_data) as blocks,
		count(extra_data) filter (where coinbase_diff_eth > 0) as blocks_profit,
		count(extra_data) filter (where coinbase_diff_eth < 0) as blocks_sub,
		round(avg(CASE WHEN coinbase_diff_eth IS NOT NULL THEN coinbase_diff_eth ELSE 0 END), 6) as avg_profit_per_block,
		round(PERCENTILE_DISC(0.5) WITHIN GROUP(ORDER BY CASE WHEN coinbase_diff_eth IS NOT NULL THEN coinbase_diff_eth ELSE 0 END), 6) as median_profit_per_block,
		round(sum(CASE WHEN coinbase_diff_eth IS NOT NULL THEN coinbase_diff_eth ELSE 0 END), 6) as total_profit,
		round(abs(sum(CASE WHEN coinbase_diff_eth < 0 THEN coinbase_diff_eth ELSE 0 END)), 6) as total_subsidies
	FROM (
		SELECT distinct(slot), extra_data, coinbase_diff_eth FROM mainnet_data_api_payload_delivered WHERE inserted_at > $1 AND inserted_at < $2
	) AS x
	GROUP BY extra_data
	ORDER BY blocks DESC;`
	err = s.DB.Select(&res, query, since.UTC(), until.UTC())
	return res, err
}

func (s *DatabaseService) GetStatsForTimerange(since, until time.Time, relay string) (relays []*TopRelayEntry, builders []*TopBuilderEntry, err error) {
	relays, err = s.GetTopRelays(since, until)
	if err != nil {
		return nil, nil, err
	}
	builders, err = s.GetTopBuilders(since, until, relay)
	if err != nil {
		return nil, nil, err
	}
	return relays, builders, nil
}
