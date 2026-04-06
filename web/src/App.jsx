import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./context/AuthProvider";
import { useAuth } from "./context/useAuth";
import Login from "./pages/Login";
import PassengerDashboard from "./pages/PassengerDashboard";
import StaffDashboard from "./pages/StaffDashboard";

function HomeRedirect() {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  if (user.role === "staff") return <Navigate to="/staff" replace />;
  return <Navigate to="/passenger" replace />;
}

function StaffRoute({ children }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  if (user.role !== "staff") return <Navigate to="/passenger" replace />;
  return children;
}

function PassengerRoute({ children }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  if (user.role !== "passenger") return <Navigate to="/staff" replace />;
  return children;
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<HomeRedirect />} />
          <Route path="/login" element={<Login />} />
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
