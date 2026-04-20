import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./context/AuthProvider";
import { useAuth } from "./context/useAuth";
import Home from "./pages/Home";
import PassengerLayout from "./pages/PassengerLayout";
import PassengerChatPage from "./pages/PassengerChatPage";
import PassengerClaimsPage from "./pages/PassengerClaimsPage";
import PassengerReportsPage from "./pages/PassengerReportsPage";
import PassengerSignInPage from "./pages/PassengerSignInPage";
import StaffAuthPage from "./pages/StaffAuthPage";
import StaffDashboard from "./pages/StaffDashboard";

function StaffRoute({ children }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/staff/auth" replace />;
  if (user.role !== "staff") return <Navigate to="/passenger/chat" replace />;
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
          <Route
            path="/passenger"
            element={
              <PassengerRoute>
                <PassengerLayout />
              </PassengerRoute>
            }
          >
            <Route index element={<Navigate to="/passenger/chat" replace />} />
            <Route path="chat" element={<PassengerChatPage />} />
            <Route path="reports" element={<PassengerReportsPage />} />
            <Route path="claims" element={<PassengerClaimsPage />} />
          </Route>
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
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
