# Google OAuth Setup

## Prerequisites
1. A Google Cloud Platform (GCP) account
2. A GCP project

## Steps to Set Up Google OAuth

### 1. Create OAuth 2.0 Credentials
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Select your project (or create a new one)
3. Navigate to **APIs & Services** > **Credentials**
4. Click **+ CREATE CREDENTIALS** > **OAuth client ID**
5. If prompted, configure the OAuth consent screen:
   - Choose **External** (unless you have a Google Workspace)
   - Fill in the required fields (App name, User support email, Developer contact)
   - Add scopes: `email`, `profile`
   - Add test users if needed (for testing before verification)
6. For Application type, select **Web application**
7. Configure:
   - **Name**: CodeStandoff (or your preferred name)
   - **Authorized JavaScript origins**: 
     - `http://localhost:8080`
   - **Authorized redirect URIs**:
     - `http://localhost:8080/auth/google/callback`
8. Click **Create**
9. Copy the **Client ID** and **Client Secret**

### 2. Set Environment Variables
Add these to your environment or `.env` file:

```bash
GOOGLE_CLIENT_ID=your-client-id-here.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret-here
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
```

### 3. Run Database Migration
Execute the migration to add the `google_id` column:

```sql
-- Run this in your PostgreSQL database (via DBeaver or psql)
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
```

Or run the migration file:
```bash
psql -U thunderbird -d codestandoff -f migrations/002_add_google_id.sql
```

### 4. Test the OAuth Flow
1. Start your backend server
2. Navigate to the signup page
3. Click the "Google" button
4. You should be redirected to Google's consent screen
5. After authorization, you'll be redirected back and logged in

## Notes
- For production, update the authorized origins and redirect URIs to your production domain
- The OAuth consent screen needs to be verified by Google for public use
- For local development, you can add test users in the OAuth consent screen settings


