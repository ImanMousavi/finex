package queries

type ReleaseCommissionQueries struct {
	TimeFrom int64 `query:"time_from" validate:"uint"`
	TimeTo   int64 `query:"time_to" validate:"uint"`
}
