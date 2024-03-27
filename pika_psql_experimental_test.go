package pika

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestF_1(t *testing.T) {
	psql := newPsql(t)
	createTestEntries(t, psql)

	m, err := Q[simpleModel1](psql).F("title", "Test").All(context.Background())
	require.Nil(t, err)
	require.NotNil(t, m)
	require.Equal(t, 1, len(m))
	require.Equal(t, "Test", m[0].Title)
}

func TestF_2(t *testing.T) {
	psql := newPsql(t)
	createTestEntries(t, psql)

	m, err := Q[simpleModel1](psql).F("title", "Test", "description", "Test").All(context.Background())
	require.Nil(t, err)
	require.NotNil(t, m)
	require.Equal(t, 1, len(m))
	require.Equal(t, "Test", m[0].Title)
}

func TestF_3Or(t *testing.T) {
	psql := newPsql(t)
	createTestEntries(t, psql)

	m, err := Q[simpleModel1](psql).F("title", "Test", "title__or", "Test2").All(context.Background())
	require.Nil(t, err)
	require.NotNil(t, m)
	require.Equal(t, 2, len(m))
	require.Equal(t, "Test", m[0].Title)
	require.Equal(t, "Test2", m[1].Title)
}

func TestU(t *testing.T) {
	psql := newPsql(t)
	createTestEntries(t, psql)

	m, err := Q[simpleModel1](psql).F("title", "Test").All(context.Background())
	require.Nil(t, err)
	require.NotNil(t, m)
	require.Equal(t, 1, len(m))
	require.Equal(t, "Test", m[0].Title)

	elem := m[0]
	elem.Title = "TestUpdated"
	err = Q[simpleModel1](psql).U(context.Background(), elem)
	require.Nil(t, err)

	m, err = Q[simpleModel1](psql).F("title", "TestUpdated").All(context.Background())
	require.Nil(t, err)
	require.NotNil(t, m)
	require.Equal(t, 1, len(m))
	require.Equal(t, "TestUpdated", m[0].Title)
}

func TestD(t *testing.T) {
	psql := newPsql(t)
	createTestEntries(t, psql)

	m, err := Q[simpleModel1](psql).F("title", "Test").All(context.Background())
	require.Nil(t, err)
	require.NotNil(t, m)
	require.Equal(t, 1, len(m))
	require.Equal(t, "Test", m[0].Title)

	elem := m[0]
	err = Q[simpleModel1](psql).D(context.Background(), elem)
	require.Nil(t, err)

	m, err = Q[simpleModel1](psql).F("title", "Test").All(context.Background())
	require.Nil(t, err)
	require.Equal(t, 0, len(m))
}
