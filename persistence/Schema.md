- Database: Records
- -------------------------------------------
- The following tables define the Records and the Versions for those Records. 
- Each Version can have multiple properties with appropriate values
- The last Version for a given Record is also the currently Active version
- --------------------------------------------------
- This table contains the actual instances of the entities defined in the 
  - record
      - id
      - createdAt
      - updatedAt
- Each row of this table represents a Version of a row in the Record table.
- When the Record is first created, a corresponding Version row is also created. 
- So, if the system is in a correct state, we should have atleast one Version row for a given Record row
    - recordVersion
        - versionId
        - recordId
        - createdAt
- For each recordVersion row, the various Fields with respective values 
    - recordVersionField
      - fieldId
      - recordVersionId
      - key
      - value
