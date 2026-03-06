package models

import (
	"testing"
)

func TestUserData_DefaultValues(t *testing.T) {
	user := &UserData{
		Level: 1,
	}

	if user.TotalEXP != 0 {
		t.Errorf("Expected TotalEXP to be 0, got %d", user.TotalEXP)
	}

	if user.CorrectAnswers != 0 {
		t.Errorf("Expected CorrectAnswers to be 0, got %d", user.CorrectAnswers)
	}

	if user.WrongAnswers != 0 {
		t.Errorf("Expected WrongAnswers to be 0, got %d", user.WrongAnswers)
	}

	if len(user.AskedQuestions) != 0 {
		t.Errorf("Expected AskedQuestions to be empty, got %v", user.AskedQuestions)
	}

	if len(user.InterviewAsked) != 0 {
		t.Errorf("Expected InterviewAsked to be empty, got %v", user.InterviewAsked)
	}
}

func TestUserData_LevelCalculation(t *testing.T) {
	tests := []struct {
		exp      int
		expected int
	}{
		{0, 1},
		{50, 1},
		{99, 1},
		{100, 2},
		{150, 2},
		{199, 2},
		{200, 3},
		{999, 10},
		{1000, 11},
	}

	for _, tt := range tests {
		level := int(tt.exp/100) + 1
		if level != tt.expected {
			t.Errorf("EXP %d: expected level %d, got %d", tt.exp, tt.expected, level)
		}
	}
}

func TestQuestion_JSONTags(t *testing.T) {
	q := Question{
		ID:       1,
		Question: "Test question?",
		Options:  []string{"A", "B", "C", "D"},
		Correct:  0,
		Exp:      10,
	}

	if q.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", q.ID)
	}

	if len(q.Options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(q.Options))
	}
}

func TestUsersMap(t *testing.T) {
	users := make(UsersMap)

	chatID := int64(12345)
	user := &UserData{TotalEXP: 100}

	users[chatID] = user

	if users[chatID] != user {
		t.Error("Expected user to be stored in map")
	}

	if _, exists := users[99999]; exists {
		t.Error("Expected non-existent user to not exist")
	}
}
