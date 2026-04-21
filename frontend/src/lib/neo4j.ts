import neo4j, { Driver, QueryResult } from 'neo4j-driver'

const URI = import.meta.env.VITE_NEO4J_URI ?? 'bolt://localhost:7687'
const USERNAME = import.meta.env.VITE_NEO4J_USERNAME ?? ''
const PASSWORD = import.meta.env.VITE_NEO4J_PASSWORD ?? ''

let driver: Driver | null = null

export function getDriver(): Driver {
  if (!driver) {
    driver = neo4j.driver(URI, neo4j.auth.basic(USERNAME, PASSWORD))
  }
  return driver
}

export async function runQuery(
  cypher: string,
  params: Record<string, unknown> = {},
): Promise<QueryResult> {
  const session = getDriver().session()
  try {
    return await session.run(cypher, params)
  } finally {
    await session.close()
  }
}

export async function closeDriver(): Promise<void> {
  if (driver) {
    await driver.close()
    driver = null
  }
}
