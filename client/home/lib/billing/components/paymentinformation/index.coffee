
React = require 'kd-react'

module.exports = class PaymentInformation extends React.Component

  render: ->

    <section className="HomeAppView--section payment">
      <fieldset>
        <label>Nickname:</label>
        <input className={inputClass()} defaultValue='Koding Visa' />
        <input className={inputClass 'hidden'} />
      </fieldset>
    </section>


inputClass = (name) -> ['kdinput', 'text', name].filter(Boolean).join ' '
