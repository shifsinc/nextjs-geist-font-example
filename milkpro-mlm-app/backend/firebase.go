package main

import (
    "context"
    firebase "firebase.google.com/go"
    "firebase.google.com/go/auth"
    "google.golang.org/api/option"
)

var firebaseAuth *auth.Client

func initFirebase() error {
    ctx := context.Background()
    
    // Initialize Firebase app with credentials
    // In production, use a service account key file
    app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile("firebase-credentials.json"))
    if err != nil {
        return err
    }

    // Initialize Firebase Auth client
    firebaseAuth, err = app.Auth(ctx)
    if err != nil {
        return err
    }

    return nil
}

func verifyFirebaseToken(tokenString string) (*auth.Token, error) {
    ctx := context.Background()
    token, err := firebaseAuth.VerifyIDToken(ctx, tokenString)
    if err != nil {
        return nil, err
    }
    return token, nil
}
