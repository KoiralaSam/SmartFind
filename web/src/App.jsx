import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./context/AuthProvider";
import { useAuth } from "./context/useAuth";
import Home from "./pages/Home";
import PassengerChatPage from "./pages/PassengerChatPage";
import PassengerDashboard from "./pages/PassengerDashboard";
import PassengerSignInPage from "./pages/PassengerSignInPage";
import StaffAuthPage from "./pages/StaffAuthPage";
import StaffDashboard from "./pages/StaffDashboard";

function StaffRoute({ children }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/staff/auth" replace />;
  if (user.role !== "staff") return <Navigate to="/passenger" replace />;
  return children;
}

function PassengerRoute({ children }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/passenger/sign-in" replace />;
  if (user.role !== "passenger") return <Navigate to="/staff" replace />;
  return children;
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/staff/auth" element={<StaffAuthPage />} />
          <Route
            path="/staff/login"
            element={<Navigate to="/staff/auth" replace />}
          />
          <Route
            path="/staff/signup"
            element={<Navigate to="/staff/auth?mode=signup" replace />}
          />
          <Route path="/passenger/sign-in" element={<PassengerSignInPage />} />
          <Route path="/passenger/chat" element={<PassengerChatPage />} />
          <Route
            path="/auth"
            element={<Navigate to="/" replace />}
          />
          <Route
            path="/staff"
            element={
              <StaffRoute>
                <StaffDashboard />
              </StaffRoute>
            }
          />
          <Route
            path="/passenger"
            element={
              <PassengerRoute>
                <PassengerDashboard />
              </PassengerRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
