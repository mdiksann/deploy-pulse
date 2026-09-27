import React, { createContext, useContext, useEffect, useMemo, useState } from "react";
import { api, setUnauthorizedHandler } from "./api";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
	const [authError, setAuthError] = useState("");

	function checkSession() {
		setLoading(true);
		setAuthError("");
		return api("/api/auth/me")
			.then((data) => setUser(data.user))
			.catch((error) => {
				if (error.status === 401) setUser(null);
				else setAuthError(error.message);
			})
			.finally(() => setLoading(false));
	}

  useEffect(() => {
    setUnauthorizedHandler(() => setUser(null));
		checkSession();
  }, []);

  const value = useMemo(() => ({
    user,
    loading,
		authError,
		checkSession,
    setUser,
    logout: async () => {
      await api("/api/auth/logout", { method: "POST" });
      setUser(null);
    },
  }), [user, loading, authError]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export const useAuth = () => useContext(AuthContext);
