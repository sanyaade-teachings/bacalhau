import React from "react"
import styles from "./Header.module.scss"
import { ReactSVG } from "react-svg"

interface HeaderProps {
  collapsed: boolean
  pageTitle: string
}

export const Header: React.FC<HeaderProps> = ({ pageTitle, collapsed }) => {
  return (
    <header className={styles.header} data-collapsed={collapsed}>
      <div className={styles.left}>
        <ReactSVG src="../../images/bacalhau.svg" height="24" width="" />
        <div className={styles.pageTitle}>{pageTitle}</div>
        <div className={styles.searchBar}>
          {/* Placeholder for search bar */}
          {/* <input type="text" placeholder="Search..." /> */}
        </div>
      </div>
      <div className={styles.right}>
        {/* Profile section */}
        <div className={styles.profile}>
          <ReactSVG src="../../images/bacalhau.svg" />
        </div>
      </div>
    </header>
  )
}
