import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import TorrentCard from 'components/TorrentCard'
import CircularProgress from '@material-ui/core/CircularProgress'
import TextField from '@material-ui/core/TextField'
import SearchIcon from '@material-ui/icons/Search'
import { TorrentListWrapper, CenteredGrid } from 'components/App/style'
import OfflineToolbar from 'components/OfflineToolbar'
import { useOfflineFilter } from 'components/OfflineToolbar/useOfflineFilter'
import { filterTorrents } from 'utils/offline'
import AddFirstTorrent from './AddFirstTorrent'
import { SearchFieldWrapper } from './style'
import NoServerConnection from './NoServerConnection'

export default function TorrentList({ isOffline, isLoading, sortABC, torrents, sortCategory }) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const { summary, hasOffline, filter, setFilter } = useOfflineFilter(torrents)

  const sortedTorrents = useMemo(() => {
    if (!torrents) return []
    const query = search.trim().toLowerCase()
    const filtered = filterTorrents(
      torrents.filter(torrent =>
        (sortCategory === 'all' || torrent.category === sortCategory) &&
        (!query || [torrent.title, torrent.name].some(name => (name || '').toLowerCase().includes(query)))
      ),
      filter,
    )

    if (sortABC) {
      return [...filtered].sort((a, b) => (a.title || '').localeCompare(b.title || '') || a.hash.localeCompare(b.hash))
    }

    return [...filtered].sort((a, b) => {
      const tsA = a.timestamp || 0
      const tsB = b.timestamp || 0
      if (tsA !== tsB) return tsB - tsA
      return a.hash.localeCompare(b.hash)
    })
  }, [torrents, sortCategory, sortABC, filter, search])

  if (isLoading || isOffline || !torrents?.length) {
    return (
      <CenteredGrid>
        {isOffline ? (
          <NoServerConnection />
        ) : isLoading ? (
          <CircularProgress color='secondary' />
        ) : (
          !torrents.length && <AddFirstTorrent />
        )}
      </CenteredGrid>
    )
  }

  return (
    <TorrentListWrapper>
      {hasOffline && (
        <div style={{ gridColumn: '1 / -1' }}>
          <OfflineToolbar torrents={torrents} summary={summary} filter={filter} onFilterChange={setFilter} />
        </div>
      )}
      <SearchFieldWrapper>
        <SearchIcon color='action' />
        <TextField
          id='torrent-search'
          type='search'
          label={t('Search')}
          value={search}
          onChange={event => setSearch(event.target.value)}
          variant='outlined'
          size='small'
          fullWidth
        />
      </SearchFieldWrapper>
      {sortedTorrents.map(torrent => (
        <TorrentCard key={torrent.hash} torrent={torrent} />
      ))}
    </TorrentListWrapper>
  )
}