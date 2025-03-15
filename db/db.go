package db

import (
	"database/sql"
	"fmt"
	"log"
	"rugby-live-api/models"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Store struct {
	DB *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{
		DB: db,
	}
}

func (s *Store) UpsertCountry(country *models.Country) error {
	query := `
        INSERT INTO countries (code, name, flag, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $4)
        ON CONFLICT (code) 
        DO UPDATE SET
            name = EXCLUDED.name,
            flag = EXCLUDED.flag,
            updated_at = EXCLUDED.updated_at
        RETURNING code`

	return s.DB.QueryRow(
		query,
		country.Code,
		country.Name,
		country.Flag,
		time.Now(),
	).Scan(&country.Code)
}

func (s *Store) UpsertLeague(league *models.League) error {
	query := `
        INSERT INTO leagues (
            id, 
            name, 
            type, 
            logo, 
            country_code,
            created_at, 
            updated_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $6)
        ON CONFLICT (id)
        DO UPDATE SET
            name = EXCLUDED.name,
            type = EXCLUDED.type,
            logo = EXCLUDED.logo,
            country_code = EXCLUDED.country_code,
            updated_at = EXCLUDED.updated_at
        RETURNING id`

	// First ensure the country exists
	if err := s.UpsertCountry(&league.Country); err != nil {
		return fmt.Errorf("failed to upsert country: %v", err)
	}

	// Then create/update the league
	err := s.DB.QueryRow(
		query,
		league.ID,
		league.Name,
		league.Type,
		league.Logo,
		league.Country.Code, // Use country code as foreign key
		time.Now(),
	).Scan(&league.ID)

	if err != nil {
		return err
	}

	// Insert seasons if they exist
	if len(league.Seasons) > 0 {
		for _, season := range league.Seasons {
			if err := s.UpsertSeason(league.ID, &season); err != nil {
				log.Printf("Error upserting season for league %s: %v", league.ID, err)
			}
		}
	}

	return nil
}

// Add this new function to handle seasons
func (s *Store) UpsertSeason(leagueID string, season *models.Season) error {
	query := `
        INSERT INTO seasons (
            league_id,
            year,
            current,
            start_date,
            end_date,
            created_at,
            updated_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $6)
        ON CONFLICT (league_id, year)
        DO UPDATE SET
            current = EXCLUDED.current,
            start_date = EXCLUDED.start_date,
            end_date = EXCLUDED.end_date,
            updated_at = EXCLUDED.updated_at`

	_, err := s.DB.Exec(
		query,
		leagueID,
		season.Year,
		season.Current,
		season.Start,
		season.End,
		time.Now(),
	)

	return err
}

func (s *Store) UpsertTeam(team *models.Team) error {
	// First ensure the team exists
	if err := s.UpsertCountry(&team.Country); err != nil {
		return fmt.Errorf("failed to upsert team country: %v", err)
	}
	print("Upserting team:")
	print(team)
	// Convert string slice to Postgres array
	var altNames interface{}
	if team.AltNames != nil {
		altNames = pq.Array(team.AltNames)
	} else {
		altNames = pq.Array([]string{})
	}

	query := `
        INSERT INTO teams (id, name, country_code, logo_url, logo_source, alternate_names, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
        ON CONFLICT (id)
        DO UPDATE SET
            name = EXCLUDED.name,
            logo_url = EXCLUDED.logo_url,
            logo_source = EXCLUDED.logo_source,
            alternate_names = EXCLUDED.alternate_names,
            updated_at = EXCLUDED.updated_at
        RETURNING id`

	return s.DB.QueryRow(
		query,
		team.ID,
		team.Name,
		team.Country.Code,
		team.LogoURL,
		team.LogoSource,
		altNames,
		time.Now(),
	).Scan(&team.ID)
}

func (s *Store) UpsertMatch(match *models.Match) error {
	query := `
        INSERT INTO matches (
            id, home_team_id, away_team_id, league_id,
            home_score, away_score, status, kick_off,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
        ON CONFLICT (id)
        DO UPDATE SET
            home_score = EXCLUDED.home_score,
            away_score = EXCLUDED.away_score,
            status = EXCLUDED.status,
            updated_at = EXCLUDED.updated_at
        RETURNING id`

	return s.DB.QueryRow(
		query,
		match.ID,
		match.HomeTeamID,
		match.AwayTeamID,
		match.LeagueID,
		match.HomeScore,
		match.AwayScore,
		match.Status,
		match.KickOff,
		time.Now(),
	).Scan(&match.ID)
}

func (s *Store) UpsertMatchAPIMapping(mapping *models.MatchAPIMapping) error {
	query := `
        INSERT INTO api_mappings (entity_id, api_name, api_id, entity_type, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $5)
        ON CONFLICT (api_name, api_id)
        DO UPDATE SET
            entity_id = EXCLUDED.entity_id,
            updated_at = EXCLUDED.updated_at
        RETURNING id`

	return s.DB.QueryRow(
		query,
		mapping.MatchID,
		mapping.APIName,
		mapping.APIMatchID,
		"match",
		time.Now(),
	).Scan(&mapping.ID)
}

func (s *Store) UpsertAPIMapping(mapping *models.APIMapping) error {
	query := `
        INSERT INTO api_mappings (api_name, api_id, entity_type, entity_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $5)
        ON CONFLICT (api_name, api_id, entity_type) 
        DO UPDATE SET
            entity_id = EXCLUDED.entity_id,
            updated_at = EXCLUDED.updated_at`

	_, err := s.DB.Exec(
		query,
		mapping.APIName,
		mapping.APIID,
		mapping.EntityType,
		mapping.EntityID,
		time.Now(),
	)
	return err
}

func (s *Store) GetCountries() ([]models.Country, error) {
	query := `
        SELECT code, name, flag, created_at, updated_at 
        FROM countries 
        ORDER BY name`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []models.Country
	for rows.Next() {
		var c models.Country
		if err := rows.Scan(&c.Code, &c.Name, &c.Flag, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		countries = append(countries, c)
	}
	return countries, nil
}

func (s *Store) GetAPIMappingsByEntityType(entityType string) ([]models.APIMapping, error) {
	query := `
        SELECT entity_id, api_name, api_id, entity_type, created_at, updated_at
        FROM api_mappings
        WHERE entity_type = $1`

	rows, err := s.DB.Query(query, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []models.APIMapping
	for rows.Next() {
		var m models.APIMapping
		if err := rows.Scan(&m.EntityID, &m.APIName, &m.APIID, &m.EntityType, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, nil
}

func (s *Store) UpsertStadium(stadium *models.Stadium) error {
	query := `
        INSERT INTO stadiums (
            id,
            name,
            capacity,
            location,
            country_code,
            created_at,
            updated_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $6)
        ON CONFLICT (id)
        DO UPDATE SET
            name = EXCLUDED.name,
            capacity = EXCLUDED.capacity,
            location = EXCLUDED.location,
            country_code = EXCLUDED.country_code,
            updated_at = EXCLUDED.updated_at
        RETURNING id`

	return s.DB.QueryRow(
		query,
		stadium.ID,
		stadium.Name,
		stadium.Capacity,
		stadium.Location,
		stadium.Country.Code,
		time.Now(),
	).Scan(&stadium.ID)
}

func (s *Store) GetCountryByCode(code string) (*models.Country, error) {
	query := `
        SELECT code, name, flag, created_at, updated_at 
        FROM countries 
        WHERE code = $1`

	var country models.Country
	err := s.DB.QueryRow(query, code).Scan(
		&country.Code,
		&country.Name,
		&country.Flag,
		&country.CreatedAt,
		&country.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error querying country: %v", err)
	}
	return &country, nil
}

func (s *Store) GetLeagueByID(id string) (*models.League, error) {
	// First get the league
	query := `
        SELECT id, name, type, logo, logo_source, country_code, created_at, updated_at 
        FROM leagues 
        WHERE id = $1`

	var league models.League
	err := s.DB.QueryRow(query, id).Scan(
		&league.ID,
		&league.Name,
		&league.Type,
		&league.Logo,
		&league.LogoSource,
		&league.Country.Code,
		&league.CreatedAt,
		&league.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error querying league: %v", err)
	}

	// Then get the seasons
	seasonsQuery := `
        SELECT year, current, start_date, end_date
        FROM seasons
        WHERE league_id = $1
        ORDER BY year`

	rows, err := s.DB.Query(seasonsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error querying seasons: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var season models.Season
		err := rows.Scan(&season.Year, &season.Current, &season.Start, &season.End)
		if err != nil {
			return nil, fmt.Errorf("error scanning season: %v", err)
		}
		league.Seasons = append(league.Seasons, season)
	}

	return &league, nil
}

func (s *Store) GetTeamByID(id string) (*models.Team, error) {
	query := `
        SELECT id, name, logo_url, logo_source, country_code, created_at, updated_at, alternate_names 
        FROM teams 
        WHERE id = $1`

	var team models.Team
	err := s.DB.QueryRow(query, id).Scan(
		&team.ID,
		&team.Name,
		&team.LogoURL,
		&team.LogoSource,
		&team.Country.Code,
		&team.CreatedAt,
		&team.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error querying team: %v", err)
	}
	return &team, nil
}

func (s *Store) GetAPIMappingByAPIID(apiName string, apiID string, entityType string) (*models.APIMapping, error) {
	query := `
        SELECT entity_id, api_name, api_id, entity_type, created_at, updated_at
        FROM api_mappings
        WHERE api_name = $1 AND api_id = $2 AND entity_type = $3`

	var mapping models.APIMapping
	err := s.DB.QueryRow(query, apiName, apiID, entityType).Scan(
		&mapping.EntityID,
		&mapping.APIName,
		&mapping.APIID,
		&mapping.EntityType,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying API mapping: %v", err)
	}
	return &mapping, nil
}

func (s *Store) UpsertTeamStadium(teamID string, stadium *models.TeamStadium) error {
	query := `
        INSERT INTO team_stadiums (
            team_id,
            stadium_id,
            is_primary,
            start_date,
            end_date,
            created_at,
            updated_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $6)
        ON CONFLICT (team_id, stadium_id) 
        DO UPDATE SET
            is_primary = EXCLUDED.is_primary,
            start_date = EXCLUDED.start_date,
            end_date = EXCLUDED.end_date,
            updated_at = EXCLUDED.updated_at`

	_, err := s.DB.Exec(
		query,
		teamID,
		stadium.Stadium.ID,
		stadium.IsPrimary,
		stadium.StartDate,
		stadium.EndDate,
		time.Now(),
	)
	return err
}

func (s *Store) GetAllTeams() ([]*models.Team, error) {
	query := `
        SELECT t.id, t.name, t.logo_url, t.logo_source, t.created_at, t.updated_at,
        c.code as country_code, c.name as country_name, c.flag as country_flag
        FROM teams t
        JOIN countries c ON t.country_code = c.code
        ORDER BY t.name`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		var logoURL, logoSource sql.NullString
		team := &models.Team{}
		err := rows.Scan(
			&team.ID,
			&team.Name,
			&logoURL,
			&logoSource,
			&team.CreatedAt,
			&team.UpdatedAt,
			&team.Country.Code,
			&team.Country.Name,
			&team.Country.Flag,
		)
		if err != nil {
			return nil, err
		}
		team.LogoURL = logoURL.String
		team.LogoSource = logoSource.String
		teams = append(teams, team)
	}
	return teams, nil
}

func (s *Store) GetAPIMappingsByType(apiName string, entityType string) ([]models.APIMapping, error) {
	query := `
		SELECT 
			id,
			entity_id,
			api_name,
			api_id,
			entity_type,
			created_at,
			updated_at
		FROM api_mappings 
		WHERE api_name = $1 AND entity_type = $2
	`
	var mappings []models.APIMapping
	rows, err := s.DB.Query(query, apiName, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m models.APIMapping
		err := rows.Scan(
			&m.ID,
			&m.EntityID,
			&m.APIName,
			&m.APIID,
			&m.EntityType,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, err
}

func (s *Store) GetTeamsByCountryCode(countryCode string) ([]*models.Team, error) {
	query := `
        SELECT t.id, t.name, t.logo_url, t.logo_source, t.created_at, t.updated_at,
        c.code as country_code, c.name as country_name, c.flag as country_flag
        FROM teams t
        JOIN countries c ON t.country_code = c.code
        WHERE t.country_code = $1
        ORDER BY t.name`

	rows, err := s.DB.Query(query, countryCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		var logoURL, logoSource sql.NullString
		team := &models.Team{}
		err := rows.Scan(
			&team.ID,
			&team.Name,
			&logoURL,
			&logoSource,
			&team.CreatedAt,
			&team.UpdatedAt,
			&team.Country.Code,
			&team.Country.Name,
			&team.Country.Flag,
		)
		if err != nil {
			return nil, err
		}
		team.LogoURL = logoURL.String
		team.LogoSource = logoSource.String
		teams = append(teams, team)
	}
	return teams, nil
}

func (s *Store) GetCountryByName(name string) (*models.Country, error) {
	var country models.Country
	err := s.DB.QueryRow("SELECT code, name, flag FROM countries WHERE name = $1", name).Scan(
		&country.Code,
		&country.Name,
		&country.Flag,
	)
	if err != nil {
		return nil, err
	}
	return &country, nil
}
