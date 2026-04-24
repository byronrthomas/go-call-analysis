import { Routes, Route } from 'react-router-dom'
import { HomePage } from './pages/HomePage'
import { ExplorePage } from './pages/ExplorePage'
import './App.css'

export default function App() {
  return (
    <div className="app">
      <header>
        <h1>Go Call Analysis</h1>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/explore" element={<ExplorePage />} />
        </Routes>
      </main>
    </div>
  )
}
